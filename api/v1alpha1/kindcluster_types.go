/*
Copyright 2022.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type ControlPlaneEndpoint struct {
	Host string `json:"host,omitempty" yaml:"host,omitempty"`
	Port int    `json:"port,omitempty" yaml:"port,omitempty"`
}

// KindClusterSpec defines the desired state of KindCluster
type KindClusterSpec struct {
	ControlPlaneEndpoint     `json:"controlPlaneEndpoint,omitempty" yaml:"controlPlaneEndpoint,omitempty"`
	WorkerMachineCount       int    `json:"workerMachineCount,omitempty" yaml:"workerMachineCount,omitempty"`
	ControlPlaneMachineCount int    `json:"controlPlaneMachineCount,omitempty" yaml:"controlPlaneMachineCount,omitempty"`
	KubernetesVersion        string `json:"kubernetesVersion,omitempty" yaml:"kubernetesVersion,omitempty"`
}

// KindClusterStatus defines the observed state of KindCluster
type KindClusterStatus struct {
	Ready          bool   `json:"ready,omitempty" yaml:"ready,omitempty"`
	FailureReason  string `json:"failureReason,omitempty" yaml:"failureReason,omitempty"`
	FailureMessage string `json:"failureMessage,omitempty" yaml:"failureMessage,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KindCluster is the Schema for the kindclusters API
type KindCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KindClusterSpec   `json:"spec,omitempty"`
	Status KindClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KindClusterList contains a list of KindCluster
type KindClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KindCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KindCluster{}, &KindClusterList{})
}
