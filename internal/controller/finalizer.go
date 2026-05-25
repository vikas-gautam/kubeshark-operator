package controller

import (
	"context"

	kubesharkv1beta1 "kubeshark-operator/api/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// reconcileDelete runs when the CR is being deleted. Namespaced children are removed by
// owner references; cluster-scoped RBAC must be deleted explicitly.
func (r *KubesharkReconciler) reconcileDelete(ctx context.Context, cr *kubesharkv1beta1.Kubeshark) error {
	if !controllerutil.ContainsFinalizer(cr, finalizerName) {
		return nil
	}

	clusterObjects := []client.Object{
		&rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: clusterRoleName(cr)}},
		&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: clusterRoleBindingName(cr)}},
	}
	for _, obj := range clusterObjects {
		if err := r.Delete(ctx, obj); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}

	controllerutil.RemoveFinalizer(cr, finalizerName)
	return r.Update(ctx, cr)
}
