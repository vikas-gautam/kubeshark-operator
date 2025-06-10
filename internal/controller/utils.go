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
			Name:      getOrDefaultString(cr.Spec.HubServiceName, "kubeshark-hub"),
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
			Name:      getOrDefaultString(cr.Spec.HubConfigMapName, "kubeshark-config-map"),
			Namespace: cr.Namespace,
		},
	}

	defaultData := map[string]string{
		"POD_REGEX":                        ".*",
		"NAMESPACES":                       "",
		"EXCLUDED_NAMESPACES":              "",
		"BPF_OVERRIDE":                     "",
		"STOPPED":                          "false",
		"SCRIPTING_SCRIPTS":                "{}",
		"SCRIPTING_ACTIVE_SCRIPTS":         "",
		"INGRESS_ENABLED":                  "false",
		"INGRESS_HOST":                     "ks.svc.cluster.local",
		"PROXY_FRONT_PORT":                 "8899",
		"AUTH_ENABLED":                     "true",
		"AUTH_TYPE":                        "default",
		"AUTH_SAML_IDP_METADATA_URL":       "",
		"AUTH_SAML_ROLE_ATTRIBUTE":         "role",
		"AUTH_SAML_ROLES":                  `{"admin":{"canDownloadPCAP":true,"canStopTrafficCapturing":true,"canUpdateTargetedPods":true,"canUseScripting":true,"filter":"","scriptingPermissions":{"canActivate":true,"canDelete":true,"canSave":true},"showAdminConsoleLink":true}}`,
		"AUTH_OIDC_ISSUER":                 "not set",
		"AUTH_OIDC_REFRESH_TOKEN_LIFETIME": "3960h",
		"AUTH_OIDC_STATE_PARAM_EXPIRY":     "10m",
		"AUTH_OIDC_BYPASS_SSL_CA_CHECK":    "false",
		"TELEMETRY_DISABLED":               "false",
		"SCRIPTING_DISABLED":               "false",
		"TARGETED_PODS_UPDATE_DISABLED":    "",
		"PRESET_FILTERS_CHANGING_ENABLED":  "true",
		"RECORDING_DISABLED":               "",
		"STOP_TRAFFIC_CAPTURING_DISABLED":  "false",
		"GLOBAL_FILTER":                    "",
		"DEFAULT_FILTER":                   "!dns and !error",
		"TRAFFIC_SAMPLE_RATE":              "100",
		"JSON_TTL":                         "5m",
		"PCAP_TTL":                         "10s",
		"PCAP_ERROR_TTL":                   "60s",
		"TIMEZONE":                         " ",
		"CLOUD_LICENSE_ENABLED":            "true",
		"AI_ASSISTANT_ENABLED":             "true",
		"DUPLICATE_TIMEFRAME":              "200ms",
		"ENABLED_DISSECTORS":               "amqp,dns,http,icmp,kafka,redis,sctp,ws,ldap,radius,diameter",
		"CUSTOM_MACROS":                    `{"https":"tls and (http or http2)"}`,
		"DISSECTORS_UPDATING_ENABLED":      "true",
		"DETECT_DUPLICATES":                "false",
		"PCAP_DUMP_ENABLE":                 "true",
		"PCAP_TIME_INTERVAL":               "1m",
		"PCAP_MAX_TIME":                    "1h",
		"PCAP_MAX_SIZE":                    "500MB",
		"PORT_MAPPING":                     `{"amqp":[5671,5672],"diameter":[3868],"http":[80,443,8080],"kafka":[9092],"ldap":[389],"redis":[6379]}`,
	}

	op, err := controllerutil.CreateOrUpdate(context.Background(), r.Client, desiredConfigMap, func() error {
		desiredConfigMap.Labels = labels

		// Use custom data if provided, otherwise use default
		if cr.Spec.HubConfigMapData != nil {
			desiredConfigMap.Data = cr.Spec.HubConfigMapData
		} else {
			desiredConfigMap.Data = defaultData
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
				Name:      getOrDefaultString(cr.Spec.ServiceAccountHub, "kubeshark-service-account"), // Dynamic service account name from CR or default
				Namespace: cr.Namespace,
			},
			{
				Kind:      "ServiceAccount",
				Name:      getOrDefaultString(cr.Spec.ServiceAccountWorker, "kubeshark-worker"), // Dynamic service account name from CR or default
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

func (r *KubesharkReconciler) createOrUpdateWorkerDaemonSet(
	cr *kubesharkv1beta1.Kubeshark,
	labels map[string]string,
) (*appsv1.DaemonSet, string, error) {

	namespace := cr.Spec.Namespace
	if namespace == "" {
		namespace = cr.Namespace // fallback to the CR's namespace
	}

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubeshark-worker-daemonset",
			Namespace: namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, ds, func() error {
		ds.Labels = labels
		ds.Spec = buildWorkerDaemonSetSpec(cr, labels)
		return nil
	})
	if err != nil {
		return nil, "", err
	}

	return ds, string(op), nil
}

func buildWorkerDaemonSetSpec(cr *kubesharkv1beta1.Kubeshark, labels map[string]string) appsv1.DaemonSetSpec {
	privileged := true
	bidirectional := corev1.MountPropagationBidirectional
	serviceAccount := cr.Spec.ServiceAccountWorker
	if serviceAccount == "" {
		serviceAccount = "kubeshark-worker"
	}

	return appsv1.DaemonSetSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{
					{
						Name:            "mount-bpf",
						Image:           cr.Spec.WorkerImage,
						ImagePullPolicy: corev1.PullAlways,
						Command: []string{
							"/bin/sh", "-c",
							"mkdir -p /sys/fs/bpf && mount | grep -q '/sys/fs/bpf' || mount -t bpf bpf /sys/fs/bpf",
						},
						SecurityContext: &corev1.SecurityContext{
							Privileged: &privileged,
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:             "sys",
								MountPath:        "/sys",
								MountPropagation: &bidirectional,
							},
						},
					},
				},
				Containers: []corev1.Container{
					buildSnifferContainer(cr),
					buildTracerContainer(cr),
				},
				ServiceAccountName: serviceAccount,
				HostNetwork:        true,
				DNSPolicy:          corev1.DNSClusterFirstWithHostNet,
				Tolerations: []corev1.Toleration{
					{Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoExecute},
				},
				Affinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{
								{
									MatchExpressions: []corev1.NodeSelectorRequirement{
										{
											Key:      "kubernetes.io/os",
											Operator: corev1.NodeSelectorOpIn,
											Values:   []string{"linux"},
										},
									},
								},
							},
						},
					},
				},
				Volumes: buildWorkerVolumes(),
			},
		},
	}
}
func buildSnifferContainer(cr *kubesharkv1beta1.Kubeshark) corev1.Container {
	privileged := true

	return corev1.Container{
		Name:            "sniffer",
		Image:           cr.Spec.WorkerImage,
		ImagePullPolicy: corev1.PullAlways,
		Command: []string{
			"./worker", "-i", "any", "-port", "48999", "-metrics-port", "49100",
			"-packet-capture", "best", "warning",
			"-servicemesh", "-procfs", "/hostproc",
			"-enable-watchdog", "-resolution-strategy", "auto", "-staletimeout", "30",
		},
		Ports: []corev1.ContainerPort{
			{Name: "metrics", ContainerPort: 49100, Protocol: corev1.ProtocolTCP},
		},
		Env:       getCommonEnv(),
		Resources: getDefaultResources(),
		SecurityContext: &corev1.SecurityContext{
			Privileged: &privileged,
		},
		ReadinessProbe: getTCPProbe(48999),
		LivenessProbe:  getTCPProbe(48999),
		VolumeMounts: getCommonVolumeMounts([]corev1.VolumeMount{
			{Name: "data", MountPath: "/app/sniffer_data"}, // Unique path for sniffer,
		}),
	}
}

func getTCPProbe(port int32) *corev1.Probe {
	return &corev1.Probe{
		InitialDelaySeconds: 5,
		PeriodSeconds:       5,
		FailureThreshold:    3,
		SuccessThreshold:    1,
		TimeoutSeconds:      1,
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(int(port)),
			},
		},
	}
}

func buildTracerContainer(cr *kubesharkv1beta1.Kubeshark) corev1.Container {
	privileged := true
	hostToContainer := corev1.MountPropagationHostToContainer

	return corev1.Container{
		Name:            "tracer",
		Image:           cr.Spec.WorkerImage,
		ImagePullPolicy: corev1.PullAlways,
		Command: []string{
			"./tracer", "-procfs", "/hostproc", "-disable-tls-log", "warning",
		},
		Env:       getCommonEnv(),
		Resources: getDefaultResources(),
		SecurityContext: &corev1.SecurityContext{
			Privileged: &privileged,
		},
		VolumeMounts: getCommonVolumeMounts([]corev1.VolumeMount{
			{Name: "data", MountPath: "/app/tracer_data"}, // Unique path for tracer
			{Name: "os-release", MountPath: "/etc/os-release", ReadOnly: true},
			{Name: "root", MountPath: "/hostroot", ReadOnly: true, MountPropagation: &hostToContainer},
		}),
	}
}
func buildWorkerVolumes() []corev1.Volume {
	return []corev1.Volume{
		{Name: "proc", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/proc"}}},
		{Name: "sys", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/sys"}}},
		{Name: "lib-modules", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/lib/modules"}}},
		{Name: "os-release", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/etc/os-release"}}},
		{Name: "root", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/"}}},
		{Name: "data", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{
			SizeLimit: resource.NewQuantity(5*1024*1024*1024, resource.BinarySI),
		}}},
	}
}
func getCommonEnv() []corev1.EnvVar {
	return []corev1.EnvVar{
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
		{
			Name:  "PROFILING_ENABLED",
			Value: "false",
		},
		{
			Name:  "SENTRY_ENABLED",
			Value: "false",
		},
		{
			Name:  "SENTRY_ENVIRONMENT",
			Value: "production",
		},
	}
}
func getDefaultResources() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("5Gi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("50Mi"),
		},
	}
}
func getCommonVolumeMounts(extraMounts []corev1.VolumeMount) []corev1.VolumeMount {
	commonMounts := []corev1.VolumeMount{
		{
			Name:      "proc",
			MountPath: "/hostproc",
			ReadOnly:  true,
		},
		{
			Name:      "sys",
			MountPath: "/sys",
			ReadOnly:  true,
			MountPropagation: func() *corev1.MountPropagationMode {
				mode := corev1.MountPropagationHostToContainer
				return &mode
			}(),
		},
		{
			Name:      "data",
			MountPath: "/app/data",
		},
	}
	return append(commonMounts, extraMounts...)
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

func (r *KubesharkReconciler) createOrUpdateKubesharkSecret(cr *kubesharkv1beta1.Kubeshark, labels map[string]string) (*corev1.Secret, string, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubeshark-secret",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
	}

	op, err := controllerutil.CreateOrUpdate(context.Background(), r.Client, secret, func() error {
		if secret.StringData == nil {
			secret.StringData = map[string]string{}
		}

		secret.Type = corev1.SecretTypeOpaque

		// Accessing fields from nested SecretKubeshark struct
		sk := cr.Spec.SecretKubeshark

		secret.StringData["LICENSE"] = getOrDefaultString(sk.License, "")
		secret.StringData["SCRIPTING_ENV"] = getOrDefaultString(sk.ScriptingEnvJson, "{}")
		secret.StringData["OIDC_CLIENT_ID"] = getOrDefaultString(sk.OidcClientID, "not set")
		secret.StringData["OIDC_CLIENT_SECRET"] = getOrDefaultString(sk.OidcClientSecret, "not set")

		return nil
	})

	if err != nil {
		return nil, "", fmt.Errorf("failed to create or update Secret: %w", err)
	}

	resp := fmt.Sprintf("%s kubeshark-secret", op)
	return secret, resp, nil
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
