/*
Copyright 2024.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KubesharkSpec defines the desired state of Kubeshark
type KubesharkSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	HubImage               string                 `json:"hubImage,omitempty"`
	FrontendImage          string                 `json:"frontendImage,omitempty"`
	HubReplicas            int32                  `json:"hubReplicas,omitempty"`
	WorkerImage            string                 `json:"workerImage,omitempty"`
	Namespace              string                 `json:"namespace,omitempty"`
	IngressHost            string                 `json:"ingressHost,omitempty"`
	ServiceAccountHub      string                 `json:"serviceAccountHub,omitempty"`
	ServiceAccountWorker   string                 `json:"serviceAccountWorker,omitempty"`
	HubConfigMapData       map[string]string      `json:"hubConfigMapData,omitempty"`
	ClusterRoleRules       []rbacv1.PolicyRule    `json:"clusterRoleRules,omitempty"`
	ClusterRoleName        string                 `json:"clusterRoleName,omitempty"`
	ClusterRoleBindingName string                 `json:"clusterRoleBindingName,omitempty"`
	HubDeploymentName      string                 `json:"hubDeploymentName,omitempty"`
	HubLabels              map[string]string      `json:"hubLabels,omitempty"`
	HubContainerPorts      []corev1.ContainerPort `json:"hubContainerPorts,omitempty"`
	WorkerDaemonSetName    string                 `json:"workerDaemonSetName,omitempty"`
	WorkerLabels           map[string]string      `json:"workerLabels,omitempty"`
	HubServiceName         string                 `json:"hubServiceName,omitempty"`
	IngressName            string                 `json:"ingressName,omitempty"`
	IngressClassName       string                 `json:"ingressClassName,omitempty"`
	HubConfigMapName       string                 `json:"hubConfigMapName,omitempty"`
}

// KubesharkStatus defines the observed state of Kubeshark
type KubesharkStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Phase indicates the current phase of the resource
	Phase string `json:"phase,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Kubeshark is the Schema for the kubesharks API
type Kubeshark struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KubesharkSpec   `json:"spec,omitempty"`
	Status KubesharkStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KubesharkList contains a list of Kubeshark
type KubesharkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kubeshark `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Kubeshark{}, &KubesharkList{})
}
