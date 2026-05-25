package controller

import (
	"context"
	"fmt"

	kubesharkv1beta1 "kubeshark-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *KubesharkReconciler) reconcileRBAC(ctx context.Context, cr *kubesharkv1beta1.Kubeshark, labels map[string]string) error {
	if err := r.reconcileWorkerServiceAccount(ctx, cr, labels); err != nil {
		return err
	}
	if err := r.reconcileClusterRole(ctx, cr, labels); err != nil {
		return err
	}
	return r.reconcileClusterRoleBinding(ctx, cr, labels)
}

func defaultClusterRoleRules() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"configmaps", "secrets", "nodes", "pods", "services", "endpoints", "persistentvolumeclaims"},
			Verbs:     []string{"list", "get", "watch"},
		},
		{
			APIGroups:     []string{""},
			Resources:     []string{"namespaces"},
			ResourceNames: []string{"kube-system"},
			Verbs:         []string{"get"},
		},
		{
			APIGroups: []string{"extensions", "apps"},
			Resources: []string{"deployments"},
			Verbs:     []string{"get", "list", "watch"},
		},
	}
}

func (r *KubesharkReconciler) reconcileWorkerServiceAccount(ctx context.Context, cr *kubesharkv1beta1.Kubeshark, labels map[string]string) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: workerServiceAccountName(cr), Namespace: cr.Namespace},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, sa, func() error {
		if err := controllerutil.SetControllerReference(cr, sa, r.Scheme); err != nil {
			return err
		}
		sa.Labels = labels
		return nil
	})
	return err
}

func (r *KubesharkReconciler) reconcileClusterRole(ctx context.Context, cr *kubesharkv1beta1.Kubeshark, labels map[string]string) error {
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: clusterRoleName(cr), Labels: labels},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, role, func() error {
		if len(cr.Spec.ClusterRoleRules) > 0 {
			role.Rules = cr.Spec.ClusterRoleRules
		} else {
			role.Rules = defaultClusterRoleRules()
		}
		role.Labels = labels
		return nil
	})
	if err != nil {
		return fmt.Errorf("ClusterRole: %w", err)
	}
	return nil
}

func (r *KubesharkReconciler) reconcileClusterRoleBinding(ctx context.Context, cr *kubesharkv1beta1.Kubeshark, labels map[string]string) error {
	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: clusterRoleBindingName(cr), Labels: labels},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, binding, func() error {
		binding.Labels = labels
		binding.RoleRef = rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     clusterRoleName(cr),
			APIGroup: rbacv1.GroupName,
		}
		binding.Subjects = []rbacv1.Subject{
			{Kind: rbacv1.ServiceAccountKind, Name: hubServiceAccountName(cr), Namespace: cr.Namespace},
			{Kind: rbacv1.ServiceAccountKind, Name: workerServiceAccountName(cr), Namespace: cr.Namespace},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("ClusterRoleBinding: %w", err)
	}
	return nil
}
