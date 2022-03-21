/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	zkv1alpha1 "github.com/ciiiii/zookeeper-operator/pkg/apis/v1alpha1"
	"github.com/ciiiii/zookeeper-operator/pkg/constants"
)

var (
	restoreLogger = log.Log.WithName(constants.RestoreController)
)

// ZooKeeperRestoreReconciler reconciles a ZooKeeperRestore object
type ZooKeeperRestoreReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=zookeeper.example.com,resources=zookeeperrestores,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=zookeeper.example.com,resources=zookeeperrestores/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=zookeeper.example.com,resources=zookeeperrestores/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ZooKeeperRestore object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ZooKeeperRestoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	restore := &zkv1alpha1.ZooKeeperRestore{}
	if err := r.Get(ctx, req.NamespacedName, restore); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	restoreLogger.Info("Reconciling ZooKeeperRestore", "Name", restore.Name, "Namespace", restore.Namespace)

	msg, err := r.reconcileJob(ctx, restore)
	if err != nil {
		return ctrl.Result{}, err
	}
	if msg != "" {
		return ctrl.Result{}, r.reconcileMessage(ctx, restore, msg)
	}
	return ctrl.Result{}, r.reconcileStatus(ctx, restore)
}

func (r *ZooKeeperRestoreReconciler) reconcileJob(ctx context.Context, restore *zkv1alpha1.ZooKeeperRestore) (string, error) {
	// message means update status.message
	// error will be retried
	pvcList, stroageLocation, msg, err := r.parseRequiredInfo(ctx, restore)
	if msg != "" || err != nil {
		return msg, err
	}
	if stroageLocation.Key == "" {
		return "storage location key is not specified", nil
	}
	desired := renderRestoreJob(restore, pvcList, stroageLocation)
	restoreLogger.Info("Reconciling Job", "Name", restore.Name, "Namespace", restore.Namespace)
	current := &batchv1.Job{}
	err = r.Get(ctx, client.ObjectKey{Name: desired.Name, Namespace: desired.Namespace}, current)
	switch {
	case err != nil && errors.IsNotFound(err):
		restoreLogger.Info("Creating Job", "Name", desired.Name, "Namespace", desired.Namespace)
		if err := controllerutil.SetControllerReference(restore, desired, r.Scheme); err != nil {
			return "", err
		}
		if err := r.Create(ctx, desired); err != nil {
			return "", err
		}
	case err != nil:
		return "", err
	default:
	}
	return "", nil
}

func (r *ZooKeeperRestoreReconciler) reconcileStatus(ctx context.Context, restore *zkv1alpha1.ZooKeeperRestore) error {
	restoreLogger.Info("Reconciling Status", "Name", restore.Name, "Namespace", restore.Namespace)
	targetJob := &batchv1.Job{}
	if err := r.Get(ctx, client.ObjectKey{Name: fmt.Sprintf(constants.RestorePrefix, restore.Name), Namespace: restore.Namespace}, targetJob); err != nil {
		return err
	}
	if restore.Status.Status == zkv1alpha1.RestoreStatusCompleted && !restore.Spec.RolloutRestart {
		return nil
	}
	if restore.Status.Status == zkv1alpha1.RestoreStatusRestarted && restore.Spec.RolloutRestart {
		return nil
	}
	var status zkv1alpha1.ClusterRestoreStatus
	switch {
	case targetJob.Status.Succeeded == *targetJob.Spec.Completions:
		status = zkv1alpha1.RestoreStatusCompleted
	case targetJob.Status.Failed == *targetJob.Spec.BackoffLimit+1:
		status = zkv1alpha1.RestoreStatusFailed
	default:
		for _, condition := range targetJob.Status.Conditions {
			if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
				status = zkv1alpha1.RestoreStatusFailed
				break
			}
		}
		if status != zkv1alpha1.RestoreStatusFailed {
			status = zkv1alpha1.RestoreStatusRunning
		}
	}
	if status != zkv1alpha1.RestoreStatusFailed {
		pods := &corev1.PodList{}
		selector := client.MatchingLabels(targetJob.Spec.Selector.MatchLabels)
		if err := r.List(ctx, pods, selector); err != nil {
			return err
		}
		if len(pods.Items) == 0 {
			status = zkv1alpha1.RestoreStatusPending
		} else {
			if status == "" {
				status = zkv1alpha1.RestoreStatusRunning
			}
		}
	}
	message := ""
	if restore.Status.Status != zkv1alpha1.RestoreStatusRestarted && status == zkv1alpha1.RestoreStatusCompleted && restore.Spec.RolloutRestart {
		targetSts, msg, err := restore.Spec.Target.GetStatefulSet(r.Client, restore.Namespace)
		if err != nil {
			return err
		}
		if msg != "" {
			message = msg
		} else {
			if err := r.rolloutRestart(ctx, targetSts); err != nil {
				return err
			}
			status = zkv1alpha1.RestoreStatusRestarted
		}
	}
	patch := client.MergeFrom(restore.DeepCopy())
	restore.Status.Message = message
	restore.Status.Status = status
	if err := r.Status().Patch(ctx, restore, patch); err != nil {
		return err
	}
	return nil
}

func (r *ZooKeeperRestoreReconciler) rolloutRestart(ctx context.Context, sts *appsv1.StatefulSet) error {
	restoreLogger.Info("Start RolloutRestarting StatefulSet", "Name", sts.Name, "Namespace", sts.Namespace)
	patch := client.MergeFrom(sts.DeepCopy())
	if sts.Spec.Template.Annotations == nil {
		sts.Spec.Template.Annotations = make(map[string]string)
	}
	sts.Spec.Template.Annotations[constants.RolloutRestartAnnotation] = time.Now().Format(constants.RestartTimeFormat)
	if err := r.Patch(ctx, sts, patch); err != nil {
		return err
	}
	restoreLogger.Info("RolloutRestart StatefulSet success", "Name", sts.Name, "Namespace", sts.Namespace)
	return nil
}

func (r *ZooKeeperRestoreReconciler) reconcileMessage(ctx context.Context, restore *zkv1alpha1.ZooKeeperRestore, message string) error {
	restoreLogger.Info("Reconciling Status message", "Name", restore.Name, "Namespace", restore.Namespace, "Message", message)
	patch := client.MergeFrom(restore.DeepCopy())
	restore.Status.Message = message
	if err := r.Status().Patch(ctx, restore, patch); err != nil {
		return err
	}
	return nil
}

func (r *ZooKeeperRestoreReconciler) parseRequiredInfo(ctx context.Context, restore *zkv1alpha1.ZooKeeperRestore) ([]string, *zkv1alpha1.GenericStroageLocation, string, error) {
	// message means update status.message
	// error will be retried
	sts, msg, err := restore.Spec.Target.GetStatefulSet(r.Client, restore.Namespace)
	if msg != "" || err != nil {
		return nil, nil, msg, err
	}
	if len(sts.Spec.VolumeClaimTemplates) == 0 {
		return nil, nil, fmt.Sprintf("no PVC bind with target StatefulSet %s:%s", sts.Namespace, sts.Name), nil
	}
	pvcNamePrefix := sts.Spec.VolumeClaimTemplates[0].Name
	var pvcList []string
	for index := 0; index < int(*sts.Spec.Replicas); index++ {
		pvcName := fmt.Sprintf("%s-%s-%d", pvcNamePrefix, sts.Name, index)
		pvcList = append(pvcList, pvcName)
	}
	stroageLocation, err := restore.Spec.Source.GetStroageLocation()
	if err != nil {
		return nil, nil, err.Error(), nil
	}
	return pvcList, stroageLocation, "", nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ZooKeeperRestoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New(constants.RestoreController, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err := c.Watch(
		&source.Kind{Type: &zkv1alpha1.ZooKeeperRestore{}},
		&handler.EnqueueRequestForObject{},
		predicate.GenerationChangedPredicate{}); err != nil {
		return err
	}

	if err := c.Watch(
		&source.Kind{Type: &batchv1.Job{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &zkv1alpha1.ZooKeeperRestore{},
		}); err != nil {
		return err
	}

	return nil
}
