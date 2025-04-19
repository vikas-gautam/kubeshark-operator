package controller

import (
	"context"
	"fmt"
	kubesharkv1beta1 "kubeshark-operator/api/v1beta1"

	networkingv1 "k8s.io/api/networking/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *KubesharkReconciler) createOrUpdateHubService(cr *kubesharkv1beta1.Kubeshark, labels map[string]string) (*corev1.Service, string, error) {
	desiredService := &corev1.Service{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      getOrDefaultString(cr.Spec.HubServiceName, "kubeshark-hub-service"),
			Namespace: cr.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(context.Background(), r.Client, desiredService, func() error {
		desiredService.Labels = labels
		desiredService.Spec.Ports = []corev1.ServicePort{
			{
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			},
		}
		desiredService.Spec.Selector = labels
		return nil
	})

	if err != nil {
		return nil, "", err
	}

	resp := fmt.Sprintf("%s Service:", op)
	return desiredService, resp, nil
}

func (r *KubesharkReconciler) createOrUpdateHubDeployment(cr *kubesharkv1beta1.Kubeshark, labels map[string]string) (*appsv1.Deployment, string, error) {
	desiredDeployment := &appsv1.Deployment{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      getOrDefaultString(cr.Spec.HubDeploymentName, "kubeshark-hub-deployment"),
			Namespace: cr.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(context.Background(), r.Client, desiredDeployment, func() error {
		desiredDeployment.Spec.Replicas = getOrDefaultInt(cr.Spec.HubReplicas, 1)
		if desiredDeployment.CreationTimestamp.IsZero() {
			desiredDeployment.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: labels,
			}
		}
		desiredDeployment.Spec.Template = corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      labels,
				Annotations: desiredDeployment.Spec.Template.ObjectMeta.Annotations,
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: getOrDefaultString(cr.Spec.ServiceAccountHub, "kubeshark-service-account"),
				Containers: []corev1.Container{
					{
						Name:  "kubeshark-hub",
						Image: getOrDefaultString(cr.Spec.HubImage, "docker.io/kubeshark/hub:v52.3.82"),
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: 8080,
							},
						},
						Env: []corev1.EnvVar{
							{
								Name: "POD_NAME",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "metadata.name",
									},
								},
							},
							{
								Name: "POD_NAMESPACE",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "metadata.namespace",
									},
								},
							},
						},
					},
				},
			},
		}
		return nil
	})

	if err != nil {
		return nil, "", fmt.Errorf("failed to create or update Hub Deployment: %w", err)
	}

	resp := fmt.Sprintf("%s Deployment:", op)
	return desiredDeployment, resp, nil
}

func (r *KubesharkReconciler) createOrUpdateHubConfigMap(cr *kubesharkv1beta1.Kubeshark, labels map[string]string) (*corev1.ConfigMap, string, error) {
	desiredConfigMap := &corev1.ConfigMap{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      getOrDefaultString(cr.Spec.HubConfigMapName, "kubeshark-hub-config"),
			Namespace: cr.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(context.Background(), r.Client, desiredConfigMap, func() error {
		desiredConfigMap.Labels = labels
		if cr.Spec.HubConfigMapData != nil {
			desiredConfigMap.Data = cr.Spec.HubConfigMapData
		}
		return nil
	})

	if err != nil {
		return nil, "", err
	}

	resp := fmt.Sprintf("%s ConfigMap:", op)
	return desiredConfigMap, resp, nil
}

func (r *KubesharkReconciler) createOrUpdateServiceAccount(cr *kubesharkv1beta1.Kubeshark, labels map[string]string) (*corev1.ServiceAccount, string, error) {
	desiredServiceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubeshark-service-account",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
	}

	op, err := controllerutil.CreateOrUpdate(context.Background(), r.Client, desiredServiceAccount, func() error {
		return nil
	})

	if err != nil {
		return nil, "", err
	}

	resp := fmt.Sprintf("%s ServiceAccount", op)
	return desiredServiceAccount, resp, nil
}

func (r *KubesharkReconciler) createOrUpdateClusterRole(labels map[string]string) (*rbacv1.ClusterRole, string, error) {
	desiredClusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kubeshark-cluster-role",
			Labels: labels,
		},
	}

	// Use CreateOrUpdate to create or update the ClusterRole
	op, err := controllerutil.CreateOrUpdate(context.Background(), r.Client, desiredClusterRole, func() error {
		// Set the rules dynamically within the mutate function
		desiredClusterRole.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"nodes", "pods", "services", "endpoints", "persistentvolumeclaims"},
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

		// No need to mutate other fields; we already defined them above
		return nil
	})

	if err != nil {
		return nil, "", fmt.Errorf("failed to create or update ClusterRole: %w", err)
	}

	// Return the result of the CreateOrUpdate operation
	resp := fmt.Sprintf("%s ClusterRole", op)
	return desiredClusterRole, resp, nil
}

func (r *KubesharkReconciler) createOrUpdateClusterRoleBinding(cr *kubesharkv1beta1.Kubeshark, labels map[string]string) (*rbacv1.ClusterRoleBinding, string, error) {
	desiredClusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kubeshark-cluster-role-binding",
			Labels: labels,
		},
	}

	op, err := controllerutil.CreateOrUpdate(context.Background(), r.Client, desiredClusterRoleBinding, func() error {
		// Dynamically set the Subjects and RoleRef in mutateFn based on the cr
		desiredClusterRoleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      getOrDefaultString(cr.Spec.ServiceAccountHub, "kubeshark-sa"), // Dynamic service account name from CR or default
				Namespace: cr.Namespace,
			},
		}
		desiredClusterRoleBinding.RoleRef = rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     getOrDefaultString(cr.Spec.ClusterRoleName, "kubeshark-cluster-role"), // Dynamic role name from CR or default
			APIGroup: "rbac.authorization.k8s.io",
		}

		return nil
	})

	if err != nil {
		return nil, "", fmt.Errorf("failed to create or update ClusterRoleBinding: %w", err)
	}

	resp := fmt.Sprintf("%s ClusterRoleBinding", op)
	return desiredClusterRoleBinding, resp, nil
}

func (r *KubesharkReconciler) createOrUpdateFrontendService(cr *kubesharkv1beta1.Kubeshark, labels map[string]string) (*corev1.Service, string, error) {
	// Initial object meta setup, kept outside the mutateFn
	desiredService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubeshark-frontend-service",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
	}

	// CreateOrUpdate with mutateFn that dynamically adjusts Spec
	op, err := controllerutil.CreateOrUpdate(context.Background(), r.Client, desiredService, func() error {
		// Mutating the service Spec dynamically based on the custom resource
		desiredService.Spec.Selector = labels
		desiredService.Spec.Ports = []corev1.ServicePort{
			{
				Name:       "http",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			},
		}
		desiredService.Spec.Type = corev1.ServiceTypeClusterIP

		// You can add more dynamic logic here based on cr.Spec or other conditions
		return nil
	})

	if err != nil {
		return nil, "", fmt.Errorf("failed to create or update Frontend Service: %w", err)
	}

	resp := fmt.Sprintf("%s Frontend Service", op)
	return desiredService, resp, nil
}

func (r *KubesharkReconciler) createOrUpdateFrontendDeployment(cr *kubesharkv1beta1.Kubeshark, labels map[string]string) (*appsv1.Deployment, string, error) {
	replicas := int32(1) // Set number of replicas as needed
	desiredDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubeshark-frontend-deployment",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "frontend",
							Image: getOrDefaultString(cr.Spec.FrontendImage, "docker.io/kubeshark/front:v52.3.82"), // Default image
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8080,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "REACT_APP_DEFAULT_FILTER",
									Value: "{}",
								},
								{
									Name:  "REACT_APP_AUTH_ENABLED",
									Value: "false",
								},
								{
									Name:  "REACT_APP_AUTH_TYPE",
									Value: "none",
								},
								{
									Name:  "REACT_APP_AUTH_SAML_IDP_METADATA_URL",
									Value: "https://example.com",
								},
								{
									Name:  "REACT_APP_TIMEZONE",
									Value: "IST",
								},
								{
									Name:  "REACT_APP_REPLAY_DISABLED",
									Value: "false",
								},
								{
									Name:  "REACT_APP_RECORDING_DISABLED",
									Value: "false",
								},
								{
									Name:  "REACT_APP_SCRIPTING_DISABLED",
									Value: "false",
								},
								{
									Name:  "REACT_APP_TARGETED_PODS_UPDATE_DISABLED",
									Value: "false",
								},
								{
									Name:  "REACT_APP_BPF_OVERRIDE_DISABLED",
									Value: "false",
								},
								{
									Name:  "REACT_APP_CLOUD_LICENSE_ENABLED",
									Value: "false",
								},
								{
									Name:  "REACT_APP_STOP_TRAFFIC_CAPTURING_DISABLED",
									Value: "false",
								},
								{
									Name:  "REACT_APP_SUPPORT_CHAT_ENABLED",
									Value: "false",
								},
								{
									Name:  "REACT_APP_DISSECTORS_UPDATING_ENABLED",
									Value: "false",
								},
								{
									Name:  "REACT_APP_SENTRY_ENABLED",
									Value: "false",
								},
								// Add other environment variables as needed
							},
						},
					},
				},
			},
		},
	}

	op, err := controllerutil.CreateOrUpdate(context.Background(), r.Client, desiredDeployment, func() error {
		// Mutate function can remain empty for frontend since no dynamic changes are necessary here.
		return nil
	})

	if err != nil {
		return nil, "", fmt.Errorf("failed to create or update Frontend Deployment: %w", err)
	}

	resp := fmt.Sprintf("%s Frontend Deployment", op)
	return desiredDeployment, resp, nil
}

func (r *KubesharkReconciler) createOrUpdateFrontendIngress(cr *kubesharkv1beta1.Kubeshark, labels map[string]string) (*networkingv1.Ingress, string, error) {
	desiredIngress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubeshark-frontend-ingress",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
	}

	op, err := controllerutil.CreateOrUpdate(context.Background(), r.Client, desiredIngress, func() error {
		// Ensure labels and spec are updated
		desiredIngress.Labels = labels

		desiredIngress.Spec.IngressClassName = &cr.Spec.IngressClassName

		// Ensure spec and host are updated
		desiredIngress.Spec.Rules = []networkingv1.IngressRule{
			{
				Host: cr.Spec.IngressHost, // Explicitly set host from CR
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								Path: "/",
								PathType: func() *networkingv1.PathType {
									pt := networkingv1.PathTypeImplementationSpecific
									return &pt
								}(),
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "kubeshark-frontend-service",
										Port: networkingv1.ServiceBackendPort{
											Number: 80,
										},
									},
								},
							},
						},
					},
				},
			},
		}
		return nil
	})

	if err != nil {
		return nil, "", err
	}

	resp := fmt.Sprintf("%s Frontend Ingress", op)
	return desiredIngress, resp, nil
}

func (r *KubesharkReconciler) createOrUpdateWorkerDaemonSet(cr *kubesharkv1beta1.Kubeshark, labels map[string]string) (*appsv1.DaemonSet, string, error) {
	// ObjectMeta (static, can stay outside the mutateFn)
	desiredDaemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubeshark-worker-daemonset",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
	}

	// CreateOrUpdate with mutateFn to dynamically modify the DaemonSetSpec
	op, err := controllerutil.CreateOrUpdate(context.Background(), r.Client, desiredDaemonSet, func() error {
		// Set dynamic DaemonSet spec values inside the mutateFn
		desiredDaemonSet.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: labels,
		}

		// Set the PodTemplateSpec with dynamic values
		desiredDaemonSet.Spec.Template = corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "worker",
						Image: getOrDefaultString(cr.Spec.WorkerImage, "docker.io/kubeshark/worker:v52.3.82"),
						Env: []corev1.EnvVar{
							{
								Name:  "HUB_ADDRESS",
								Value: "http://kubeshark-hub:80", // Default hub address
							},
							{
								Name: "POD_NAME",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "metadata.name",
									},
								},
							},
							{
								Name: "POD_NAMESPACE",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "metadata.namespace",
									},
								},
							},
						},
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: 8897, // Default worker port for traffic capture
								Name:          "http",
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "host-run",
								MountPath: "/var/run", // Required for capturing traffic
							},
						},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),  // Default CPU limit
								corev1.ResourceMemory: resource.MustParse("128Mi"), // Default memory limit
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("200m"), // Default CPU request
								corev1.ResourceMemory: resource.MustParse("64Mi"), // Default memory request
							},
						},
						SecurityContext: &corev1.SecurityContext{
							Privileged: getOrDefaultBoolPtr(true), // Required for network capture
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "host-run",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/var/run", // Host path to enable network capture
							},
						},
					},
				},
				ServiceAccountName: getOrDefaultString(cr.Spec.ServiceAccountWorker, "kubeshark-worker"),
				NodeSelector: map[string]string{
					"kubernetes.io/os": "linux", // Target Linux nodes by default
				},
			},
		}
		return nil
	})

	if err != nil {
		return nil, "", fmt.Errorf("failed to create or update Worker DaemonSet: %w", err)
	}

	resp := fmt.Sprintf("%s Worker DaemonSet", op)
	return desiredDaemonSet, resp, nil
}

func (r *KubesharkReconciler) createOrUpdateWorkerServiceAccount(cr *kubesharkv1beta1.Kubeshark, labels map[string]string) (*corev1.ServiceAccount, string, error) {
	desiredServiceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getOrDefaultString(cr.Spec.ServiceAccountWorker, "kubeshark-worker"),
			Namespace: cr.Namespace,
			Labels:    labels,
		},
	}

	op, err := controllerutil.CreateOrUpdate(context.Background(), r.Client, desiredServiceAccount, func() error {
		return nil
	})

	if err != nil {
		return nil, "", fmt.Errorf("failed to create or update Worker ServiceAccount: %w", err)
	}

	resp := fmt.Sprintf("%s Worker ServiceAccount", op)
	return desiredServiceAccount, resp, nil
}

// ################################

// cleanupResource deletes a Kubernetes resource associated with the CR
func (r *KubesharkReconciler) cleanupResource(ctx context.Context, cr *kubesharkv1beta1.Kubeshark, resource client.Object) error {
	resource.SetNamespace(cr.Namespace)

	if err := r.Client.Delete(ctx, resource); err != nil && !errors.IsNotFound(err) {
		return err
	}

	return nil
}
