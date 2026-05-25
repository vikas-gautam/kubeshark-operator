package controller

import (
	"context"
	"fmt"

	kubesharkv1beta1 "kubeshark-operator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *KubesharkReconciler) reconcileHub(ctx context.Context, cr *kubesharkv1beta1.Kubeshark, labels map[string]string) error {
	if err := r.reconcileHubConfigMap(ctx, cr, labels); err != nil {
		return err
	}
	if err := r.reconcileHubServiceAccount(ctx, cr, labels); err != nil {
		return err
	}
	if err := r.reconcileHubDeployment(ctx, cr, labels); err != nil {
		return err
	}
	return r.reconcileHubService(ctx, cr, labels)
}

func defaultHubConfigMapData() map[string]string {
	return map[string]string{
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
}

func (r *KubesharkReconciler) reconcileHubConfigMap(ctx context.Context, cr *kubesharkv1beta1.Kubeshark, labels map[string]string) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: hubConfigMapName(cr), Namespace: cr.Namespace},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		if err := controllerutil.SetControllerReference(cr, cm, r.Scheme); err != nil {
			return err
		}
		cm.Labels = labels
		if cr.Spec.HubConfigMapData != nil {
			cm.Data = cr.Spec.HubConfigMapData
		} else {
			cm.Data = defaultHubConfigMapData()
		}
		return nil
	})
	return err
}

func (r *KubesharkReconciler) reconcileHubServiceAccount(ctx context.Context, cr *kubesharkv1beta1.Kubeshark, labels map[string]string) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: hubServiceAccountName(cr), Namespace: cr.Namespace},
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

func (r *KubesharkReconciler) reconcileHubDeployment(ctx context.Context, cr *kubesharkv1beta1.Kubeshark, labels map[string]string) error {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: hubDeploymentName(cr), Namespace: cr.Namespace},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, dep, func() error {
		if err := controllerutil.SetControllerReference(cr, dep, r.Scheme); err != nil {
			return err
		}
		dep.Labels = labels
		dep.Spec.Replicas = hubReplicas(cr)
		dep.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
		ports := cr.Spec.HubContainerPorts
		if len(ports) == 0 {
			ports = []corev1.ContainerPort{{ContainerPort: 8080}}
		}
		dep.Spec.Template = corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: labels},
			Spec: corev1.PodSpec{
				ServiceAccountName: hubServiceAccountName(cr),
				Containers: []corev1.Container{{
					Name:  "kubeshark-hub",
					Image: getOrDefaultString(cr.Spec.HubImage, defaultHubImage),
					Ports: ports,
					Env: []corev1.EnvVar{
						{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
						{Name: "POD_NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
					},
				}},
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("hub Deployment: %w", err)
	}
	return nil
}

func (r *KubesharkReconciler) reconcileHubService(ctx context.Context, cr *kubesharkv1beta1.Kubeshark, labels map[string]string) error {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: hubServiceName(cr), Namespace: cr.Namespace},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		if err := controllerutil.SetControllerReference(cr, svc, r.Scheme); err != nil {
			return err
		}
		svc.Labels = labels
		svc.Spec.Selector = labels
		svc.Spec.Ports = []corev1.ServicePort{{
			Port:       80,
			TargetPort: intstr.FromInt32(8080),
		}}
		return nil
	})
	return err
}
