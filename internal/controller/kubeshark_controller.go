/*
Copyright 2026.

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

package controller

import (
	"context"

	kubesharkv1beta1 "kubeshark-operator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// KubesharkReconciler reconciles a Kubeshark object.
type KubesharkReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kubeshark.opstreelabs.in,resources=kubesharks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeshark.opstreelabs.in,resources=kubesharks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubeshark.opstreelabs.in,resources=kubesharks/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments;daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services;configmaps;secrets;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=get;list;watch;create;update;patch;delete

func (r *KubesharkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var cr kubesharkv1beta1.Kubeshark
	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !cr.DeletionTimestamp.IsZero() {
		if err := r.reconcileDelete(ctx, &cr); err != nil {
			_ = r.setReadyCondition(ctx, &cr, false, "DeleteFailed", err.Error())
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(&cr, finalizerName) {
		controllerutil.AddFinalizer(&cr, finalizerName)
		if err := r.Update(ctx, &cr); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	labels := componentLabels(&cr)
	steps := []struct {
		name string
		fn   func(context.Context, *kubesharkv1beta1.Kubeshark, map[string]string) error
	}{
		{"hub", r.reconcileHub},
		{"rbac", r.reconcileRBAC},
		{"worker", r.reconcileWorker},
		{"frontend", r.reconcileFrontend},
		{"secret", r.reconcileSecret},
	}
	for _, step := range steps {
		if err := step.fn(ctx, &cr, labels); err != nil {
			logger.Error(err, "reconcile failed", "step", step.name)
			_ = r.setReadyCondition(ctx, &cr, false, "ReconcileError", err.Error())
			return ctrl.Result{}, err
		}
		logger.Info("reconciled", "step", step.name)
	}

	if err := r.setReadyCondition(ctx, &cr, true, "Reconciled", "All components reconciled"); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KubesharkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubesharkv1beta1.Kubeshark{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&networkingv1.Ingress{}).
		Named("kubeshark").
		Complete(r)
}
