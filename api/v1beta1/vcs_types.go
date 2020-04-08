/*
Copyright 2020 Dan Molik <dan@hyperspike.io>.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GitSpec struct {
	// The External Hostname to use for Ingress
	Hostname string `json:"hostname,omitempty"`
	// Ingress annotations, IE: for certs and dns
	Annotations map[string]string `json:"annotations,omitempty"`
}

type CISpec struct {
	// The External Hostname to use for Ingress
	Hostname string `json:"hostname,omitempty"`
	// Ingress annotations, IE: for certs and dns
	Annotations map[string]string `json:"annotations,omitempty"`

	// The the CI System you wish to use options are drone and agola, default: drone
	// +kubebuilder:validation:Enum=drone;agola
	System string `json:"system,omitempty"`
}

type RegistrySpec struct {
	// The External Hostname to use for Ingress
	Hostname string `json:"hostname,omitempty"`
	// Ingress annotations, IE: for certs and dns
	Annotations map[string]string `json:"annotations,omitempty"`
}

type ClairSpec struct {
	// The External Hostname to use for Ingress
	Hostname string `json:"hostname,omitempty"`
	// Ingress annotations, IE: for certs and dns
	Annotations map[string]string `json:"annotations,omitempty"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VCSSpec defines the desired state of VCS
type VCSSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Domain      string            `json:"domain,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`

	Git GitSpec `json:"git,omitempty"`

	CI CISpec `json:"ci,omitempty"`

	Registry RegistrySpec `json:"registry,omitempty"`

	Clair ClairSpec `json:"clair,omitempty"`
}

// VCSStatus defines the observed state of VCS
type VCSStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// VCS is the Schema for the vcs API
type VCS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VCSSpec   `json:"spec,omitempty"`
	Status VCSStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VCSList contains a list of VCS
type VCSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VCS `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VCS{}, &VCSList{})
}
