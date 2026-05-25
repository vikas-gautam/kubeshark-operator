/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubesharkSpec defines the desired state of Kubeshark.
type KubesharkSpec struct {
	HubImage     string `json:"hubImage,omitempty"`
	FrontendImage string `json:"frontendImage,omitempty"`

	// HubReplicas is the hub deployment replica count. Omit to use default (1). Set 0 to scale down.
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	HubReplicas *int32 `json:"hubReplicas,omitempty"`

	WorkerImage string `json:"workerImage,omitempty"`
	Namespace   string `json:"namespace,omitempty"`

	IngressHost      string `json:"ingressHost,omitempty"`
	IngressClassName string `json:"ingressClassName,omitempty"`
	IngressName      string `json:"ingressName,omitempty"`

	ServiceAccountHub    string `json:"serviceAccountHub,omitempty"`
	ServiceAccountWorker string `json:"serviceAccountWorker,omitempty"`

	HubConfigMapData map[string]string `json:"hubConfigMapData,omitempty"`

	ClusterRoleRules       []rbacv1.PolicyRule `json:"clusterRoleRules,omitempty"`
	ClusterRoleName        string              `json:"clusterRoleName,omitempty"`
	ClusterRoleBindingName string              `json:"clusterRoleBindingName,omitempty"`

	HubDeploymentName   string                 `json:"hubDeploymentName,omitempty"`
	HubLabels           map[string]string      `json:"hubLabels,omitempty"`
	HubContainerPorts   []corev1.ContainerPort `json:"hubContainerPorts,omitempty"`
	WorkerDaemonSetName string                 `json:"workerDaemonSetName,omitempty"`
	WorkerLabels        map[string]string      `json:"workerLabels,omitempty"`
	HubServiceName      string                 `json:"hubServiceName,omitempty"`
	HubConfigMapName    string                 `json:"hubConfigMapName,omitempty"`

	// SecretKubeshark holds sensitive values written into the managed Secret.
	// Prefer referencing an existing Secret via external tooling for production.
	SecretKubeshark SecretKubesharkSpec `json:"secretKubeshark,omitempty"`
}

// SecretKubesharkSpec keys for the kubeshark Secret.
type SecretKubesharkSpec struct {
	License          string `json:"license,omitempty"`
	ScriptingEnvJson string `json:"scriptingEnvJson,omitempty"`
	OidcClientID     string `json:"oidcClientID,omitempty"`
	OidcClientSecret string `json:"oidcClientSecret,omitempty"`
}

// KubesharkStatus defines the observed state of Kubeshark.
type KubesharkStatus struct {
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration reflects the generation of the spec last reconciled successfully.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Kubeshark is the Schema for the kubesharks API.
type Kubeshark struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec KubesharkSpec `json:"spec"`

	// +optional
	Status KubesharkStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// KubesharkList contains a list of Kubeshark.
type KubesharkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Kubeshark `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Kubeshark{}, &KubesharkList{})
}
