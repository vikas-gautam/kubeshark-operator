package controller

import (
	kubesharkv1beta1 "kubeshark-operator/api/v1beta1"
	"k8s.io/utils/ptr"
)

const (
	finalizerName = "kubeshark.opstreelabs.in/finalizer"

	defaultHubImage      = "docker.io/kubeshark/hub:v52.3.82"
	defaultWorkerImage   = "docker.io/kubeshark/worker:v52.3.82"
	defaultFrontendImage = "docker.io/kubeshark/front:v52.3.82"
)

func getOrDefaultString(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func hubReplicas(cr *kubesharkv1beta1.Kubeshark) *int32 {
	if cr.Spec.HubReplicas != nil {
		return cr.Spec.HubReplicas
	}
	return ptr.To(int32(1))
}

func componentLabels(cr *kubesharkv1beta1.Kubeshark) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "kubeshark",
		"app.kubernetes.io/instance":   cr.Name,
		"app.kubernetes.io/managed-by": "kubeshark-operator",
	}
}

func hubServiceName(cr *kubesharkv1beta1.Kubeshark) string {
	return getOrDefaultString(cr.Spec.HubServiceName, cr.Name+"-hub")
}

func hubDeploymentName(cr *kubesharkv1beta1.Kubeshark) string {
	return getOrDefaultString(cr.Spec.HubDeploymentName, cr.Name+"-hub")
}

func hubConfigMapName(cr *kubesharkv1beta1.Kubeshark) string {
	return getOrDefaultString(cr.Spec.HubConfigMapName, cr.Name+"-config")
}

func workerDaemonSetName(cr *kubesharkv1beta1.Kubeshark) string {
	return getOrDefaultString(cr.Spec.WorkerDaemonSetName, cr.Name+"-worker")
}

func frontendServiceName(cr *kubesharkv1beta1.Kubeshark) string {
	return cr.Name + "-frontend"
}

func frontendDeploymentName(cr *kubesharkv1beta1.Kubeshark) string {
	return cr.Name + "-frontend"
}

func frontendIngressName(cr *kubesharkv1beta1.Kubeshark) string {
	return getOrDefaultString(cr.Spec.IngressName, cr.Name+"-frontend-ingress")
}

func kubesharkSecretName(cr *kubesharkv1beta1.Kubeshark) string {
	return cr.Name + "-secret"
}

func hubServiceAccountName(cr *kubesharkv1beta1.Kubeshark) string {
	return getOrDefaultString(cr.Spec.ServiceAccountHub, cr.Name+"-hub")
}

func workerServiceAccountName(cr *kubesharkv1beta1.Kubeshark) string {
	return getOrDefaultString(cr.Spec.ServiceAccountWorker, cr.Name+"-worker")
}

func clusterRoleName(cr *kubesharkv1beta1.Kubeshark) string {
	return getOrDefaultString(cr.Spec.ClusterRoleName, cr.Namespace+"-"+cr.Name+"-kubeshark")
}

func clusterRoleBindingName(cr *kubesharkv1beta1.Kubeshark) string {
	return getOrDefaultString(cr.Spec.ClusterRoleBindingName, cr.Namespace+"-"+cr.Name+"-kubeshark")
}

func workerImage(cr *kubesharkv1beta1.Kubeshark) string {
	return getOrDefaultString(cr.Spec.WorkerImage, defaultWorkerImage)
}
