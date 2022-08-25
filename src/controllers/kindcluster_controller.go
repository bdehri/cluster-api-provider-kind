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
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrastructurev1alpha1 "github.com/bdehri/cluster-api-provider-kind/api/v1alpha1"
	"github.com/bdehri/cluster-api-provider-kind/src/kind"
	"github.com/go-logr/logr"
)

const (
	kindClusterFinalizer = "giantswarm.kindcluster/finalizer"
	requeueAfter         = time.Second * 30
)

// KindClusterReconciler reconciles a KindCluster object
type KindClusterReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	KindClient kind.ClientInt
}

type KindClusterManagerInterface interface {
}

type KindClusterManager struct {
	client.Client
	kindClient kind.ClientInt
	infrastructurev1alpha1.KindCluster
	logr.Logger
	types.NamespacedName
	*patch.Helper
	*v1beta1.Cluster
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=kindclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=kindclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=kindclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KindCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *KindClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("KindCluster reconciling", "cluster", req.Name)
	var kindCluster infrastructurev1alpha1.KindCluster
	if err := r.Get(ctx, req.NamespacedName, &kindCluster); err != nil {

		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	cluster, err := util.GetOwnerCluster(ctx, r.Client, kindCluster.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}

	if cluster == nil {
		logger.Info("Cluster Controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}

	if annotations.IsPaused(cluster, &kindCluster) {
		logger.Info("reconciliation is paused for this object")
		return ctrl.Result{Requeue: true, RequeueAfter: requeueAfter}, nil
	}

	helper, err := patch.NewHelper(&kindCluster, r.Client)
	if err != nil {
		logger.Error(err, "failed to init patch helper")
		return ctrl.Result{}, err
	}

	kindClusterManager := KindClusterManager{
		Client:         r.Client,
		kindClient:     r.KindClient,
		Logger:         logger,
		KindCluster:    kindCluster,
		NamespacedName: req.NamespacedName,
		Helper:         helper,
		Cluster:        cluster,
	}

	if !kindCluster.DeletionTimestamp.IsZero() {
		reconcileDelete(ctx, kindClusterManager)
	}

	return reconcileNormal(ctx, kindClusterManager)
}
func reconcileDelete(ctx context.Context, kindClusterManager KindClusterManager) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(&kindClusterManager.KindCluster, kindClusterFinalizer) {
		err := kindClusterManager.kindClient.Delete(kindClusterManager.NamespacedName.Name)
		if err != nil {
			return ctrl.Result{}, err
		}

		kindClusterManager.Logger.Info("External kind resource deleted", "cluster", kindClusterManager.NamespacedName)
		var kubeconfigConfigmap corev1.ConfigMap
		err = kindClusterManager.Client.Get(ctx, kindClusterManager.NamespacedName, &kubeconfigConfigmap)
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		} else if !apierrors.IsNotFound(err) {
			if err := kindClusterManager.Client.Delete(ctx, &kubeconfigConfigmap); err != nil {
				return ctrl.Result{}, err
			}
			if controllerutil.ContainsFinalizer(&kubeconfigConfigmap, kindClusterFinalizer) {
				controllerutil.RemoveFinalizer(&kubeconfigConfigmap, kindClusterFinalizer)
				if err := kindClusterManager.Client.Update(ctx, &kubeconfigConfigmap); err != nil {
					return ctrl.Result{}, err
				}
			}
			kindClusterManager.Logger.Info("KindCluster configmap was deleted", "cluster", kindClusterManager.NamespacedName)
		}
	}
	controllerutil.RemoveFinalizer(&kindClusterManager.KindCluster, kindClusterFinalizer)
	if err := kindClusterManager.Update(ctx, &kindClusterManager.KindCluster); err != nil {
		return ctrl.Result{}, err
	}
	kindClusterManager.Logger.Info("KindCluster was deleted", "cluster", kindClusterManager.NamespacedName.Name)
	return ctrl.Result{}, nil
}

func reconcileNormal(ctx context.Context, kindClusterManager KindClusterManager) (ctrl.Result, error) {
	controllerutil.AddFinalizer(&kindClusterManager.KindCluster, kindClusterFinalizer)
	kindClusterManager.Logger.Info("KindCluster add finalizer", "cluster", kindClusterManager.NamespacedName)

	err := kindClusterManager.kindClient.Create(kindClusterManager.NamespacedName.Name, kindClusterManager.KindCluster.Spec.KubernetesVersion,
		kindClusterManager.Cluster.Spec.ClusterNetwork.Pods.CIDRBlocks[0], kindClusterManager.KindCluster.Spec.ControlPlaneMachineCount,
		kindClusterManager.KindCluster.Spec.WorkerMachineCount)
	if err != nil {
		kindClusterManager.Logger.Error(err, "unable to create KindCluster")
		return ctrl.Result{}, err
	}

	kubeconfig, err := kindClusterManager.kindClient.GetKubeconfig(kindClusterManager.NamespacedName.Name)
	if err != nil {
		return ctrl.Result{}, err
	}

	kubeconfigSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: kindClusterManager.NamespacedName.Namespace,
			Name:      kindClusterManager.NamespacedName.Name,
		},
		Data: map[string][]byte{
			"kubeconfig": kubeconfig,
		},
	}

	controllerutil.AddFinalizer(&kubeconfigSecret, kindClusterFinalizer)
	if err := kindClusterManager.Client.Create(ctx, &kubeconfigSecret); err != nil && !apierrors.IsAlreadyExists(err) {
		return ctrl.Result{}, err
	}

	kindClusterManager.KindCluster.Status.Ready = true
	kindClusterManager.KindCluster.Spec.ControlPlaneEndpoint.Host = "127.0.0.1"
	kindClusterManager.KindCluster.Spec.ControlPlaneEndpoint.Port = 80
	if err := kindClusterManager.Helper.Patch(
		context.TODO(),
		&kindClusterManager.KindCluster); err != nil {
		kindClusterManager.Logger.Error(err, "unable to update kindCluster status")
		return ctrl.Result{}, err
	}
	kindClusterManager.Logger.Info("kindcluster is ready", "cluster name", kindClusterManager.NamespacedName)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KindClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1alpha1.KindCluster{}).
		Complete(r)
}
