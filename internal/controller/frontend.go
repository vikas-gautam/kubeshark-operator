package controller

import (
	"context"
	"fmt"

	kubesharkv1beta1 "kubeshark-operator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *KubesharkReconciler) reconcileFrontend(ctx context.Context, cr *kubesharkv1beta1.Kubeshark, labels map[string]string) error {
	if err := r.reconcileFrontendDeployment(ctx, cr, labels); err != nil {
		return err
	}
	if err := r.reconcileFrontendService(ctx, cr, labels); err != nil {
		return err
	}
	return r.reconcileFrontendIngress(ctx, cr, labels)
}

func (r *KubesharkReconciler) reconcileFrontendDeployment(ctx context.Context, cr *kubesharkv1beta1.Kubeshark, labels map[string]string) error {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: frontendDeploymentName(cr), Namespace: cr.Namespace},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, dep, func() error {
		if err := controllerutil.SetControllerReference(cr, dep, r.Scheme); err != nil {
			return err
		}
		dep.Labels = labels
		dep.Spec.Replicas = ptr.To(int32(1))
		dep.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
		dep.Spec.Template = corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: labels},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:  "frontend",
					Image: getOrDefaultString(cr.Spec.FrontendImage, defaultFrontendImage),
					Ports: []corev1.ContainerPort{{ContainerPort: 8080}},
					Env: []corev1.EnvVar{
						{Name: "REACT_APP_DEFAULT_FILTER", Value: "{}"},
						{Name: "REACT_APP_AUTH_ENABLED", Value: "false"},
						{Name: "REACT_APP_AUTH_TYPE", Value: "none"},
						{Name: "REACT_APP_AUTH_SAML_IDP_METADATA_URL", Value: "https://example.com"},
						{Name: "REACT_APP_TIMEZONE", Value: "IST"},
						{Name: "REACT_APP_REPLAY_DISABLED", Value: "false"},
						{Name: "REACT_APP_RECORDING_DISABLED", Value: "false"},
						{Name: "REACT_APP_SCRIPTING_DISABLED", Value: "false"},
						{Name: "REACT_APP_TARGETED_PODS_UPDATE_DISABLED", Value: "false"},
						{Name: "REACT_APP_BPF_OVERRIDE_DISABLED", Value: "false"},
						{Name: "REACT_APP_CLOUD_LICENSE_ENABLED", Value: "false"},
						{Name: "REACT_APP_STOP_TRAFFIC_CAPTURING_DISABLED", Value: "false"},
						{Name: "REACT_APP_SUPPORT_CHAT_ENABLED", Value: "false"},
						{Name: "REACT_APP_DISSECTORS_UPDATING_ENABLED", Value: "false"},
						{Name: "REACT_APP_SENTRY_ENABLED", Value: "false"},
					},
				}},
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("frontend Deployment: %w", err)
	}
	return nil
}

func (r *KubesharkReconciler) reconcileFrontendService(ctx context.Context, cr *kubesharkv1beta1.Kubeshark, labels map[string]string) error {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: frontendServiceName(cr), Namespace: cr.Namespace},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		if err := controllerutil.SetControllerReference(cr, svc, r.Scheme); err != nil {
			return err
		}
		svc.Labels = labels
		svc.Spec.Type = corev1.ServiceTypeClusterIP
		svc.Spec.Selector = labels
		svc.Spec.Ports = []corev1.ServicePort{{
			Name:       "http",
			Port:       80,
			TargetPort: intstr.FromInt32(8080),
		}}
		return nil
	})
	return err
}

func (r *KubesharkReconciler) reconcileFrontendIngress(ctx context.Context, cr *kubesharkv1beta1.Kubeshark, labels map[string]string) error {
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: frontendIngressName(cr), Namespace: cr.Namespace},
	}
	pathType := networkingv1.PathTypeImplementationSpecific
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, ing, func() error {
		if err := controllerutil.SetControllerReference(cr, ing, r.Scheme); err != nil {
			return err
		}
		ing.Labels = labels
		if cr.Spec.IngressClassName != "" {
			ing.Spec.IngressClassName = &cr.Spec.IngressClassName
		}
		ing.Spec.Rules = []networkingv1.IngressRule{{
			Host: cr.Spec.IngressHost,
			IngressRuleValue: networkingv1.IngressRuleValue{
				HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{{
						Path:     "/",
						PathType: &pathType,
						Backend: networkingv1.IngressBackend{
							Service: &networkingv1.IngressServiceBackend{
								Name: frontendServiceName(cr),
								Port: networkingv1.ServiceBackendPort{Number: 80},
							},
						},
					}},
				},
			},
		}}
		return nil
	})
	return err
}
