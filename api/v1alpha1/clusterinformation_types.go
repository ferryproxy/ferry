/*
Copyright 2021 Shiming Zhang.

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

// ClusterInformationSpec defines the desired state of ClusterInformation
type ClusterInformationSpec struct {
	Kubeconfig []byte                         `json:"kubeconfig"`
	Domain     *ClusterInformationSpecDomain  `json:"domain,omitempty"`
	Ingress    *ClusterInformationSpecIngress `json:"ingress,omitempty"`
	Egress     *ClusterInformationSpecEgress  `json:"egress,omitempty"`
}

type ClusterInformationSpecDomain struct {
	Kind DomainKind `json:"domain"`
}

type DomainKind string

const (
	DomainKindService DomainKind = "Service"
	DomainKindDNS     DomainKind = "DNS"
)

type ClusterInformationSpecIngress struct {
	metav1.TypeMeta `json:",inline,omitempty"`
	HostIPs         []string `json:"hostIPs"`
	HostPort        int32    `json:"hostPort"`
}

type ClusterInformationSpecEgress struct {
	metav1.TypeMeta  `json:",inline,omitempty"`
	ServiceName      string `json:"serviceName"`
	ServiceNamespace string `json:"serviceNamespace"`
}

// ClusterInformationStatus defines the observed state of ClusterInformation
type ClusterInformationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of ClusterInformation
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ClusterInformation is the Schema for the ClusterInformations API
type ClusterInformation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterInformationSpec   `json:"spec,omitempty"`
	Status ClusterInformationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterInformationList contains a list of ClusterInformation
type ClusterInformationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterInformation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterInformation{}, &ClusterInformationList{})
}
