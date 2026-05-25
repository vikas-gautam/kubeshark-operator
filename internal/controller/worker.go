package controller

import (
	"context"

	kubesharkv1beta1 "kubeshark-operator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *KubesharkReconciler) reconcileWorker(ctx context.Context, cr *kubesharkv1beta1.Kubeshark, labels map[string]string) error {
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: workerDaemonSetName(cr), Namespace: cr.Namespace},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, ds, func() error {
		if err := controllerutil.SetControllerReference(cr, ds, r.Scheme); err != nil {
			return err
		}
		ds.Labels = labels
		ds.Spec = buildWorkerDaemonSetSpec(cr, labels)
		return nil
	})
	return err
}

func buildWorkerDaemonSetSpec(cr *kubesharkv1beta1.Kubeshark, labels map[string]string) appsv1.DaemonSetSpec {
	privileged := true
	bidirectional := corev1.MountPropagationBidirectional
	img := workerImage(cr)

	return appsv1.DaemonSetSpec{
		Selector: &metav1.LabelSelector{MatchLabels: labels},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: labels},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{{
					Name:            "mount-bpf",
					Image:           img,
					ImagePullPolicy: corev1.PullAlways,
					Command: []string{
						"/bin/sh", "-c",
						"mkdir -p /sys/fs/bpf && mount | grep -q '/sys/fs/bpf' || mount -t bpf bpf /sys/fs/bpf",
					},
					SecurityContext: &corev1.SecurityContext{Privileged: &privileged},
					VolumeMounts: []corev1.VolumeMount{{
						Name: "sys", MountPath: "/sys", MountPropagation: &bidirectional,
					}},
				}},
				Containers: []corev1.Container{
					buildSnifferContainer(img),
					buildTracerContainer(img),
				},
				ServiceAccountName: workerServiceAccountName(cr),
				HostNetwork:        true,
				DNSPolicy:          corev1.DNSClusterFirstWithHostNet,
				Tolerations: []corev1.Toleration{
					{Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoExecute},
				},
				Affinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{{
								MatchExpressions: []corev1.NodeSelectorRequirement{{
									Key:      "kubernetes.io/os",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"linux"},
								}},
							}},
						},
					},
				},
				Volumes: buildWorkerVolumes(),
			},
		},
	}
}

func buildSnifferContainer(img string) corev1.Container {
	privileged := true
	return corev1.Container{
		Name:            "sniffer",
		Image:           img,
		ImagePullPolicy: corev1.PullAlways,
		Command: []string{
			"./worker", "-i", "any", "-port", "48999", "-metrics-port", "49100",
			"-packet-capture", "best", "warning",
			"-servicemesh", "-procfs", "/hostproc",
			"-enable-watchdog", "-resolution-strategy", "auto", "-staletimeout", "30",
		},
		Ports:           []corev1.ContainerPort{{Name: "metrics", ContainerPort: 49100, Protocol: corev1.ProtocolTCP}},
		Env:             workerCommonEnv(),
		Resources:       workerDefaultResources(),
		SecurityContext: &corev1.SecurityContext{Privileged: &privileged},
		ReadinessProbe:  tcpProbe(48999),
		LivenessProbe:   tcpProbe(48999),
		VolumeMounts: workerCommonVolumeMounts([]corev1.VolumeMount{
			{Name: "data", MountPath: "/app/sniffer_data"},
		}),
	}
}

func buildTracerContainer(img string) corev1.Container {
	privileged := true
	hostToContainer := corev1.MountPropagationHostToContainer
	return corev1.Container{
		Name:            "tracer",
		Image:           img,
		ImagePullPolicy: corev1.PullAlways,
		Command:         []string{"./tracer", "-procfs", "/hostproc", "-disable-tls-log", "warning"},
		Env:             workerCommonEnv(),
		Resources:       workerDefaultResources(),
		SecurityContext: &corev1.SecurityContext{Privileged: &privileged},
		VolumeMounts: workerCommonVolumeMounts([]corev1.VolumeMount{
			{Name: "data", MountPath: "/app/tracer_data"},
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

func workerCommonEnv() []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
		{Name: "POD_NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
		{Name: "PROFILING_ENABLED", Value: "false"},
		{Name: "SENTRY_ENABLED", Value: "false"},
		{Name: "SENTRY_ENVIRONMENT", Value: "production"},
	}
}

func workerDefaultResources() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("5Gi")},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("50Mi"),
		},
	}
}

func workerCommonVolumeMounts(extra []corev1.VolumeMount) []corev1.VolumeMount {
	hostToContainer := corev1.MountPropagationHostToContainer
	common := []corev1.VolumeMount{
		{Name: "proc", MountPath: "/hostproc", ReadOnly: true},
		{Name: "sys", MountPath: "/sys", ReadOnly: true, MountPropagation: &hostToContainer},
		{Name: "data", MountPath: "/app/data"},
	}
	return append(common, extra...)
}

func tcpProbe(port int32) *corev1.Probe {
	return &corev1.Probe{
		InitialDelaySeconds: 5,
		PeriodSeconds:       5,
		FailureThreshold:    3,
		SuccessThreshold:    1,
		TimeoutSeconds:      1,
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(port)},
		},
	}
}
