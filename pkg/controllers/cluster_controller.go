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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	log "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	zkv1alpha1 "github.com/ciiiii/zookeeper-operator/pkg/apis/v1alpha1"
	"github.com/ciiiii/zookeeper-operator/pkg/constants"
)

var (
	logger = log.Log.WithName(constants.ClusterController)
)

// ZooKeeperClusterReconciler reconciles a ZooKeeperCluster object
type ZooKeeperClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=zookeeper.example.com,resources=zookeeperclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=zookeeper.example.com,resources=zookeeperclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=zookeeper.example.com,resources=zookeeperclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ZooKeeperCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ZooKeeperClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cluster := &zkv1alpha1.ZooKeeperCluster{}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.Info("Reconciling ZooKeeperCluster", "Name", cluster.Name, "Namespace", cluster.Namespace)
	for _, desiredObj := range append(renderRBAC(cluster), renderService(cluster), renderConigMap(cluster)) {
		finalObj, err := r.reconcileGeneric(ctx, cluster, desiredObj)
		if err != nil {
			logger.Info("Reconcile Generic failed", "Error", err)
			return ctrl.Result{}, err
		}
		logger.Info(fmt.Sprintf("Reconcile %s success", finalObj.GetKind()), "Name", finalObj.GetName(), "Namespace", finalObj.GetNamespace())
	}
	if err := r.reconcileStatefulset(ctx, cluster); err != nil {
		logger.Info("Reconcile Statefulset failed", "Error", err)
		return ctrl.Result{}, err
	}
	logger.Info("Reconcile Statefulset success", "Name", cluster.Name, "Namespace", cluster.Namespace)
	if err := r.reconcileStatus(ctx, cluster); err != nil {
		logger.Info("Reconcile Status failed", "Error", err)
		return ctrl.Result{}, err
	}
	if err := r.reconcileFinalizer(ctx, cluster); err != nil {
		logger.Info("Reconcile Finalizer failed", "Error", err)
		return ctrl.Result{}, err
	}
	logger.Info("Reconcile ZooKeeperCluster success", "Name", cluster.Name, "Namespace", cluster.Namespace)
	return ctrl.Result{}, nil
}

func (r *ZooKeeperClusterReconciler) reconcileStatefulset(ctx context.Context, cluster *zkv1alpha1.ZooKeeperCluster) error {
	desired := renderStatefulset(cluster)
	current := &appsv1.StatefulSet{}
	err := r.Get(ctx, client.ObjectKey{
		Namespace: cluster.Namespace,
		Name:      desired.Name,
	}, current)
	switch {
	case err != nil && errors.IsNotFound(err):
		logger.Info("Creating StatefulSet", "Name", desired.Name, "Namespace", desired.Namespace)
		if err := controllerutil.SetControllerReference(cluster, desired, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(ctx, desired); err != nil {
			return err
		}
		return nil
	case err != nil:
		logger.Error(err, "unable to get StatefulSet", "Name", desired.Name, "Namespace", desired.Namespace)
		return err
	default:
	}
	changed := false
	if current.Spec.Replicas != nil && *current.Spec.Replicas != cluster.Spec.Replicas {
		changed = true
		logger.Info(fmt.Sprintf("StatefulSet Replicas scaled from %d to %d", *current.Spec.Replicas, *desired.Spec.Replicas), "Name", desired.Name, "Namespace", desired.Namespace)
		if *current.Spec.Replicas > *desired.Spec.Replicas {
			if err := r.markRemovalPod(ctx, cluster, *current.Spec.Replicas, *desired.Spec.Replicas); err != nil {
				return err
			}
		}
		current.Spec.Replicas = desired.Spec.Replicas
	}
	if current.Spec.Template.Spec.InitContainers[0].Image != desired.Spec.Template.Spec.InitContainers[0].Image {
		changed = true
		logger.Info("StatefulSet InitContainer image changed", "Name", desired.Name, "Namespace", desired.Namespace)
		current.Spec.Template.Spec.InitContainers[0].Image = desired.Spec.Template.Spec.InitContainers[0].Image
	}
	if current.Spec.Template.Spec.Containers[0].Image != desired.Spec.Template.Spec.Containers[0].Image {
		changed = true
		logger.Info("StatefulSet Container zookeeper image changed", "Name", desired.Name, "Namespace", desired.Namespace)
		current.Spec.Template.Spec.Containers[0].Image = desired.Spec.Template.Spec.Containers[0].Image
	}
	if current.Spec.Template.Spec.Containers[0].Image != desired.Spec.Template.Spec.Containers[0].Image {
		changed = true
		logger.Info("StatefulSet Container zk-helper image changed", "Name", desired.Name, "Namespace", desired.Namespace)
		current.Spec.Template.Spec.Containers[0].Image = desired.Spec.Template.Spec.Containers[0].Image
	}

	if changed {
		if err := r.Update(ctx, current); err != nil {
			return err
		}
	}
	return nil
}

func (r *ZooKeeperClusterReconciler) reconcileStatus(ctx context.Context, cluster *zkv1alpha1.ZooKeeperCluster) error {
	labelSelector := client.MatchingLabels{"operator": "zookeeper", "instance": cluster.Name}
	pods := &corev1.PodList{}
	if err := r.List(ctx, pods, client.InNamespace(cluster.Namespace), labelSelector); err != nil {
		return err
	}
	servers := make([]*zkv1alpha1.ZooKeeperServer, len(pods.Items))
	readyServer := int32(0)
	for i, pod := range pods.Items {
		id := pod.Annotations[constants.ServerIdAnnotation]
		ready := pod.Annotations[constants.ServerReadyAnnotation]
		mode := pod.Annotations[constants.ServerModeAnnotation]
		message := pod.Annotations[constants.ServerMessageAnnotation]
		if ready == "true" {
			readyServer++
		}
		servers[i] = &zkv1alpha1.ZooKeeperServer{
			Name:    pod.Name,
			MyId:    id,
			Mode:    mode,
			Ready:   ready,
			Message: message,
		}
	}
	patch := client.MergeFrom(cluster.DeepCopy())
	cluster.Status.Service = fmt.Sprintf("%s.%s.svc.%s", cluster.Name, cluster.Namespace, cluster.Spec.ClusterDomain)
	cluster.Status.Servers = servers
	cluster.Status.ReadyReplicas = readyServer
	cluster.Status.Replicas = cluster.Spec.Replicas
	if err := r.Status().Patch(ctx, cluster, patch); err != nil {
		return err
	}
	return nil
}

func (r *ZooKeeperClusterReconciler) reconcileFinalizer(ctx context.Context, cluster *zkv1alpha1.ZooKeeperCluster) error {
	if cluster.DeletionTimestamp.IsZero() {
		if !ContainsString(cluster.Finalizers, constants.ClearFinalizer) {
			logger.Info("Adding finalizer", "Finalizer", constants.ClearFinalizer)
			cluster.Finalizers = append(cluster.Finalizers, constants.ClearFinalizer)
			if err := r.Update(ctx, cluster); err != nil {
				return err
			}
		}
	} else {
		if ContainsString(cluster.Finalizers, constants.ClearFinalizer) {
			logger.Info("Handling finalizer", "Finalizer", constants.ClearFinalizer)
			if cluster.Spec.ClearPersistence {
				if err := r.clearPVC(ctx, cluster); err != nil {
					return err
				}
			}
			logger.Info("Removing finalizer", "Finalizer", constants.ClearFinalizer)
			cluster.Finalizers = RemoveString(cluster.Finalizers, constants.ClearFinalizer)
			if err := r.Update(ctx, cluster); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *ZooKeeperClusterReconciler) reconcileGeneric(ctx context.Context, cluster *zkv1alpha1.ZooKeeperCluster, obj runtime.Object) (*unstructured.Unstructured, error) {
	desiredObj, err := r.objectToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetControllerReference(cluster, desiredObj, r.Scheme); err != nil {
		return nil, err
	}
	currentObj := &unstructured.Unstructured{}
	currentObj.SetGroupVersionKind(desiredObj.GroupVersionKind())
	err = r.Get(ctx, client.ObjectKey{
		Namespace: desiredObj.GetNamespace(),
		Name:      desiredObj.GetName(),
	}, currentObj)
	switch {
	case err != nil && errors.IsNotFound(err):
		logger.Info(fmt.Sprintf("Creating %s", desiredObj.GetKind()), "Namespace", desiredObj.GetNamespace(), "Name", desiredObj.GetName())
		if err := r.Create(ctx, desiredObj); err != nil {
			logger.Info(fmt.Sprintf("Update %s failed", desiredObj.GetKind()), "Name", desiredObj.GetName(), "Namespace", desiredObj.GetNamespace())
			return nil, err
		}
		return desiredObj, nil
	case err != nil:
		logger.Info(fmt.Sprintf("Update %s success", desiredObj.GetKind()), "Name", desiredObj.GetName(), "Namespace", desiredObj.GetNamespace())
		return nil, err
	default:
	}
	if differGeneric(desiredObj, currentObj) {
		logger.Info(fmt.Sprintf("Changes found with %s", desiredObj.GetKind()), "Name", desiredObj.GetName(), "Namespace", desiredObj.GetNamespace())
		desiredObj.SetResourceVersion(currentObj.GetResourceVersion())
		if err := r.Update(ctx, desiredObj); err != nil {
			logger.Error(err, fmt.Sprintf("unable to update %s", desiredObj.GetKind()), "Name", desiredObj.GetName(), "Namespace", desiredObj.GetNamespace())
			return nil, err
		}
		logger.Info(fmt.Sprintf("Updating %s", desiredObj.GetKind()), "Name", desiredObj.GetName(), "Namespace", desiredObj.GetNamespace())
		return desiredObj, nil
	}
	logger.Info(fmt.Sprintf("No changes found with %s", desiredObj.GetKind()), "Name", desiredObj.GetName(), "Namespace", desiredObj.GetNamespace())
	return currentObj, nil
}

func (r *ZooKeeperClusterReconciler) clearPVC(ctx context.Context, cluster *zkv1alpha1.ZooKeeperCluster) error {
	labelSelector := client.MatchingLabels{"operator": "zookeeper", "instance": cluster.Name}
	pvcs := &corev1.PersistentVolumeClaimList{}
	if err := r.List(ctx, pvcs, client.InNamespace(cluster.Namespace), labelSelector); err != nil {
		return err
	}

	backgroud := metav1.DeletePropagationBackground
	for _, pvc := range pvcs.Items {
		if err := r.Delete(ctx, &pvc, &client.DeleteOptions{
			PropagationPolicy: &backgroud,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *ZooKeeperClusterReconciler) markRemovalPod(ctx context.Context, cluster *zkv1alpha1.ZooKeeperCluster, currentReplicas, desiredReplicas int32) error {
	for i := desiredReplicas; i < currentReplicas; i++ {
		pod := &corev1.Pod{}
		if err := r.Get(ctx, client.ObjectKey{
			Namespace: cluster.Namespace,
			Name:      fmt.Sprintf("%s-%d", cluster.Name, i),
		}, pod); err != nil {
			return err
		}
		patch := client.MergeFrom(pod.DeepCopy())
		pod.Labels[constants.RemovalPodLabel] = "true"
		if err := r.Patch(ctx, pod, patch); err != nil {
			return err
		}
		logger.Info("Marked removal Pod", "Name", pod.Name, "Namespace", pod.Namespace)
	}
	return nil
}

func (r *ZooKeeperClusterReconciler) objectToUnstructured(obj runtime.Object) (*unstructured.Unstructured, error) {
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	un := &unstructured.Unstructured{Object: m}
	gvk, err := apiutil.GVKForObject(obj, r.Scheme)
	if err != nil {
		return nil, err
	}
	un.SetGroupVersionKind(gvk)
	return un, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ZooKeeperClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New(constants.ClusterController, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}
	if err := c.Watch(
		&source.Kind{Type: &zkv1alpha1.ZooKeeperCluster{}},
		&handler.EnqueueRequestForObject{},
		predicate.GenerationChangedPredicate{}); err != nil {
		return err
	}

	if err := c.Watch(
		&source.Kind{Type: &appsv1.StatefulSet{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &zkv1alpha1.ZooKeeperCluster{},
		},
		predicate.GenerationChangedPredicate{}); err != nil {
		return err
	}

	if err := c.Watch(
		&source.Kind{Type: &corev1.Service{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &zkv1alpha1.ZooKeeperCluster{},
		}); err != nil {
		return err
	}

	if err := c.Watch(
		&source.Kind{Type: &corev1.ConfigMap{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &zkv1alpha1.ZooKeeperCluster{},
		}); err != nil {
		return err
	}

	if err := c.Watch(
		&source.Kind{Type: &corev1.Pod{}},
		handler.EnqueueRequestsFromMapFunc(
			func(o client.Object) []reconcile.Request {
				reqs := []reconcile.Request{}
				pod := o.(*corev1.Pod)
				if pod.Labels["operator"] == "zookeeper" && pod.Labels["instance"] != "" {
					reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKey{Namespace: pod.Namespace, Name: pod.Labels["instance"]}})
				}
				return reqs
			},
		)); err != nil {
		return err
	}
	return nil
}
