package controller

import (
	"context"

	kubesharkv1beta1 "kubeshark-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *KubesharkReconciler) reconcileSecret(ctx context.Context, cr *kubesharkv1beta1.Kubeshark, labels map[string]string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: kubesharkSecretName(cr), Namespace: cr.Namespace},
	}
	sk := cr.Spec.SecretKubeshark
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		if err := controllerutil.SetControllerReference(cr, secret, r.Scheme); err != nil {
			return err
		}
		secret.Labels = labels
		secret.Type = corev1.SecretTypeOpaque
		if secret.StringData == nil {
			secret.StringData = map[string]string{}
		}
		secret.StringData["LICENSE"] = getOrDefaultString(sk.License, "")
		secret.StringData["SCRIPTING_ENV"] = getOrDefaultString(sk.ScriptingEnvJson, "{}")
		secret.StringData["OIDC_CLIENT_ID"] = getOrDefaultString(sk.OidcClientID, "not set")
		secret.StringData["OIDC_CLIENT_SECRET"] = getOrDefaultString(sk.OidcClientSecret, "not set")
		return nil
	})
	return err
}
