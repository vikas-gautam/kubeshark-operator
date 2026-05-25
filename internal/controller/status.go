package controller

import (
	"context"
	"fmt"

	kubesharkv1beta1 "kubeshark-operator/api/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *KubesharkReconciler) setReadyCondition(ctx context.Context, cr *kubesharkv1beta1.Kubeshark, ready bool, reason, message string) error {
	orig := cr.DeepCopy()

	status := metav1.ConditionFalse
	if ready {
		status = metav1.ConditionTrue
	}
	meta.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: cr.Generation,
	})
	cr.Status.ObservedGeneration = cr.Generation

	if err := r.Status().Patch(ctx, cr, client.MergeFrom(orig)); err != nil {
		return fmt.Errorf("patch status: %w", err)
	}
	return nil
}
