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
	"path/filepath"
	"reflect"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	"github.com/ciiiii/zookeeper-operator/pkg/client/zookeeper"
	"github.com/ciiiii/zookeeper-operator/pkg/constants"
)

var (
	backupLogger = log.Log.WithName(constants.BackupController)
)

// ZooKeeperBackupReconciler reconciles a ZooKeeperBackup object
type ZooKeeperBackupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=zookeeper.example.com,resources=zookeeperbackups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=zookeeper.example.com,resources=zookeeperbackups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=zookeeper.example.com,resources=zookeeperbackups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ZooKeeperBackup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ZooKeeperBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	backup := &zkv1alpha1.ZooKeeperBackup{}
	if err := r.Get(ctx, req.NamespacedName, backup); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	backupLogger.Info("Reconciling ZooKeeperBackup", "Name", backup.Name, "Namespace", backup.Namespace)

	if backup.Spec.Mode == zkv1alpha1.BackupOnce {
		msg, err := r.reconcileJob(ctx, backup)
		if err != nil {
			return ctrl.Result{}, err
		}
		if msg != "" {
			return ctrl.Result{}, r.reconcileMessage(ctx, backup, msg)
		} else {
			return ctrl.Result{}, r.reconcileStatusByJob(ctx, backup)
		}
	} else {
		msg, err := r.reconcileCronJob(ctx, backup)
		if err != nil {
			return ctrl.Result{}, err
		}
		if msg != "" {
			return ctrl.Result{}, r.reconcileMessage(ctx, backup, msg)
		} else {
			return ctrl.Result{}, r.reconcileStatusByCronJob(ctx, backup)
		}
	}
}
func (r *ZooKeeperBackupReconciler) reconcileJob(ctx context.Context, backup *zkv1alpha1.ZooKeeperBackup) (string, error) {
	targetPVC, stroageLocation, msg, err := r.parseRequiredInfo(ctx, backup)
	if msg != "" || err != nil {
		return msg, err
	}
	stroageLocation.Key = filepath.Join(backup.Namespace, backup.Name, string(zkv1alpha1.BackupOnce))
	desired := renderJob(backup, targetPVC, stroageLocation)
	backupLogger.Info("Reconciling Job", "Name", desired.Name, "Namespace", desired.Namespace)
	current := &batchv1.Job{}
	err = r.Get(ctx, client.ObjectKey{Name: desired.Name, Namespace: desired.Namespace}, current)
	switch {
	case err != nil && errors.IsNotFound(err):
		backupLogger.Info("Creating Job", "Name", desired.Name, "Namespace", desired.Namespace)
		if err := controllerutil.SetControllerReference(backup, desired, r.Scheme); err != nil {
			return "", err
		}
		if err := r.Create(ctx, desired); err != nil {
			return "", err
		}
		return "", nil
	case err != nil:
		return "", err
	default:
	}
	return "", nil
}

func (r *ZooKeeperBackupReconciler) reconcileCronJob(ctx context.Context, backup *zkv1alpha1.ZooKeeperBackup) (string, error) {
	targetPVC, stroageLocation, msg, err := r.parseRequiredInfo(ctx, backup)
	if msg != "" || err != nil {
		return msg, err
	}
	stroageLocation.Key = filepath.Join(backup.Namespace, backup.Name, string(zkv1alpha1.BackupSchedule))
	desired := renderCronJob(backup, targetPVC, stroageLocation)
	backupLogger.Info("Reconciling CronJob", "Name", desired.Name, "Namespace", desired.Namespace)
	current := &batchv1beta1.CronJob{}
	err = r.Get(ctx, client.ObjectKey{Name: desired.Name, Namespace: desired.Namespace}, current)
	switch {
	case err != nil && errors.IsNotFound(err):
		backupLogger.Info("Creating CronJob", "Name", desired.Name, "Namespace", desired.Namespace)
		if err := controllerutil.SetControllerReference(backup, desired, r.Scheme); err != nil {
			return "", err
		}
		if err := r.Create(ctx, desired); err != nil {
			return "", err
		}
		return "", nil
	case err != nil:
		return "", err
	default:
	}
	changed := false
	if current.Spec.Schedule != desired.Spec.Schedule {
		current.Spec.Schedule = desired.Spec.Schedule
		changed = true
	}
	if current.Spec.Suspend != desired.Spec.Suspend {
		current.Spec.Suspend = desired.Spec.Suspend
		changed = true
	}
	if current.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image != desired.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image {
		current.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image = desired.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image
		changed = true
	}
	if !reflect.DeepEqual(current.Spec.JobTemplate.Spec.Template.Spec.Volumes, desired.Spec.JobTemplate.Spec.Template.Spec.Volumes) {
		current.Spec.JobTemplate.Spec.Template.Spec.Volumes = desired.Spec.JobTemplate.Spec.Template.Spec.Volumes
		changed = true
	}
	if !reflect.DeepEqual(current.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env, desired.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env) {
		current.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env = desired.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
		changed = true
	}
	if changed {
		backupLogger.Info("Updating CronJob", "Name", desired.Name, "Namespace", desired.Namespace)
		if err := r.Update(ctx, current); err != nil {
			return "", err
		}
	}
	return "", nil
}

func (r *ZooKeeperBackupReconciler) reconcileMessage(ctx context.Context, backup *zkv1alpha1.ZooKeeperBackup, message string) error {
	backupLogger.Info("Reconciling Status message", "Name", backup.Name, "Namespace", backup.Namespace, "Message", message)
	patch := client.MergeFrom(backup.DeepCopy())
	backup.Status.Message = message
	if err := r.Status().Patch(ctx, backup, patch); err != nil {
		return err
	}
	return nil
}

func (r *ZooKeeperBackupReconciler) reconcileStatusByJob(ctx context.Context, backup *zkv1alpha1.ZooKeeperBackup) error {
	backupLogger.Info("Reconciling Status By Job", "Name", backup.Name, "Namespace", backup.Namespace)
	targetJob := &batchv1.Job{}
	if err := r.Get(ctx, client.ObjectKey{Name: fmt.Sprintf(constants.BackupPrefix, backup.Name), Namespace: backup.Namespace}, targetJob); err != nil {
		return err
	}
	var status zkv1alpha1.ClusterBackupStatus
	switch {
	case targetJob.Status.Succeeded == *targetJob.Spec.Completions:
		status = zkv1alpha1.BackupStatusCompleted
	case targetJob.Status.Failed == *targetJob.Spec.BackoffLimit+1:
		status = zkv1alpha1.BackupStatusFailed
	default:
		for _, condition := range targetJob.Status.Conditions {
			if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
				status = zkv1alpha1.BackupStatusFailed
				break
			}
		}
		if status != zkv1alpha1.BackupStatusFailed {
			status = zkv1alpha1.BackupStatusRunning
		}
	}
	records := backup.Status.Records
	if status != zkv1alpha1.BackupStatusFailed {
		pods := &corev1.PodList{}
		selector := client.MatchingLabels(targetJob.Spec.Selector.MatchLabels)
		if err := r.List(ctx, pods, client.InNamespace(backup.Namespace), selector); err != nil {
			return err
		}
		if len(pods.Items) == 0 {
			status = zkv1alpha1.BackupStatusPending
		} else {
			if status == "" {
				status = zkv1alpha1.BackupStatusRunning
			}
			for _, pod := range pods.Items {
				key := filepath.Join(backup.Namespace, backup.Name, string(zkv1alpha1.BackupOnce), pod.Name)
				var finishedTime metav1.Time
				for _, status := range pod.Status.ContainerStatuses {
					if status.State.Terminated != nil {
						finishedTime = status.State.Terminated.FinishedAt
					}
				}
				found := false
				for index, record := range backup.Status.Records {
					if key == record.Key {
						records[index].FinshedTime = finishedTime
						found = true
					}
				}
				if !found {
					records = append(records, zkv1alpha1.BackupRecord{
						Key:         key,
						StartedTime: pod.CreationTimestamp,
						FinshedTime: finishedTime,
					})
				}
			}
		}
	}
	// FIXME: patch not work for metav1.Time in array
	// ref1: https://github.com/kubernetes/kubernetes/issues/86811
	// ref2: https://github.com/kubernetes/kubernetes/pull/95423
	patch := client.MergeFrom(backup.DeepCopy())
	backup.Status.Status = status
	backup.Status.Records = records
	if err := r.Status().Patch(ctx, backup, patch); err != nil {
		return err
	}
	return nil
}

func (r *ZooKeeperBackupReconciler) reconcileStatusByCronJob(ctx context.Context, backup *zkv1alpha1.ZooKeeperBackup) error {
	backupLogger.Info("Reconciling Status By CronJob", "Name", backup.Name, "Namespace", backup.Namespace)
	targetCronJob := &batchv1beta1.CronJob{}
	if err := r.Get(ctx, client.ObjectKey{Name: fmt.Sprintf(constants.BackupPrefix, backup.Name), Namespace: backup.Namespace}, targetCronJob); err != nil {
		return err
	}
	jobs := &batchv1.JobList{}
	selector := client.MatchingLabels{
		"mode":     "schedule-backup",
		"template": targetCronJob.Name,
	}
	var status zkv1alpha1.ClusterBackupStatus
	if err := r.List(ctx, jobs, client.InNamespace(backup.Namespace), selector); err != nil {
		return err
	}
	switch {
	case backup.Spec.Suspend:
		status = zkv1alpha1.BackupStatusSuspend
	case len(jobs.Items) == 0:
		status = zkv1alpha1.BackupStatusPending
	case len(targetCronJob.Status.Active) > 0:
		status = zkv1alpha1.BackupStatusRunning
	default:
		status = zkv1alpha1.BackupStatusPending
	}

	records := backup.Status.Records
	if len(jobs.Items) > 0 {
		for _, job := range jobs.Items {
			pods := &corev1.PodList{}
			selector := client.MatchingLabels(job.Spec.Selector.MatchLabels)
			if err := r.List(ctx, pods, client.InNamespace(backup.Namespace), selector); err != nil {
				return err
			}
			for _, pod := range pods.Items {
				key := filepath.Join(backup.Namespace, backup.Name, string(zkv1alpha1.BackupSchedule), pod.Name)
				var finishedTime metav1.Time
				for _, status := range pod.Status.ContainerStatuses {
					if status.State.Terminated != nil {
						finishedTime = status.State.Terminated.FinishedAt
					}
				}
				found := false
				for index, record := range backup.Status.Records {
					if key == record.Key {
						records[index].FinshedTime = finishedTime
						found = true
					}
				}
				if !found {
					records = append(records, zkv1alpha1.BackupRecord{
						Key:         key,
						StartedTime: pod.CreationTimestamp,
						FinshedTime: finishedTime,
					})
				}
			}
		}
	}
	patch := client.MergeFrom(backup.DeepCopy())
	backup.Status.Status = status
	backup.Status.Records = records
	if err := r.Status().Patch(ctx, backup, patch); err != nil {
		return err
	}
	return nil
}

func (r *ZooKeeperBackupReconciler) parseRequiredInfo(ctx context.Context, backup *zkv1alpha1.ZooKeeperBackup) (string, *zkv1alpha1.GenericStroageLocation, string, error) {
	sts, msg, err := backup.Spec.Source.GetStatefulSet(r.Client, backup.Namespace)
	if msg != "" || err != nil {
		return "", nil, msg, err
	}
	if len(sts.Spec.VolumeClaimTemplates) == 0 {
		return "", nil, fmt.Sprintf("no PVC bind with target StatefulSet %s:%s", sts.Namespace, sts.Name), err
	}
	pvcNamePrefix := sts.Spec.VolumeClaimTemplates[0].Name
	if backup.Spec.Source.Host == "" || backup.Spec.Source.Port == 0 {
		return "", nil, "no host or port specified for backup source", err
	}
	leaderId, err := zookeeper.GetLeader(backup.Spec.Source.Host, fmt.Sprint(backup.Spec.Source.Port))
	if err != nil {
		return "", nil, "", err
	}
	if leaderId < 1 {
		return "", nil, "", fmt.Errorf("cann't get leader id")
	}
	targetPVC := fmt.Sprintf("%s-%s-%d", pvcNamePrefix, sts.Name, leaderId-1)

	stroageLocation, err := backup.Spec.Target.GetStroageLocation()
	if err != nil {
		return "", nil, err.Error(), nil
	}
	return targetPVC, stroageLocation, "", nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ZooKeeperBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New(constants.BackupController, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err := c.Watch(
		&source.Kind{Type: &zkv1alpha1.ZooKeeperBackup{}},
		&handler.EnqueueRequestForObject{},
		predicate.GenerationChangedPredicate{}); err != nil {
		return err
	}

	if err := c.Watch(
		&source.Kind{Type: &batchv1.Job{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &zkv1alpha1.ZooKeeperBackup{},
		}); err != nil {
		return err
	}

	if err := c.Watch(
		&source.Kind{Type: &batchv1beta1.CronJob{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &zkv1alpha1.ZooKeeperBackup{},
		}); err != nil {
		return err
	}
	return nil
}
