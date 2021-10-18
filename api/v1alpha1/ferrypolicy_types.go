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

// FerryPolicySpec defines the desired state of FerryPolicy
type FerryPolicySpec struct {
	Rules []FerryPolicySpecRule `json:"rules"`
}

// FerryPolicySpecRule defines the desired rule of FerryPolicyRule
type FerryPolicySpecRule struct {
	// ExportsMatch is a list of strings that are used to match the exports of the FerryPolicy.
	ExportsMatch []*Match `json:"exportsMatch,omitempty"`
	// Exports is a list of exports of the FerryPolicy.
	Exports []FerryPolicySpecRuleExport `json:"exports"`
	// ImportsMatch is a list of strings that are used to match the imports of the FerryPolicy.
	ImportsMatch []*Match `json:"importsMatch,omitempty"`
	// Imports is a list of imports of the FerryPolicy.
	Imports []FerryPolicySpecRuleImport `json:"imports"`
}

// FerryPolicySpecRuleExport defines the desired export of FerryPolicyRule
type FerryPolicySpecRuleExport struct {
	ClusterName string   `json:"clusterName"`
	Match       []*Match `json:"match,omitempty"`
}

// FerryPolicySpecRuleImport defines the desired import of FerryPolicyRule
type FerryPolicySpecRuleImport struct {
	ClusterName string   `json:"clusterName"`
	Match       []*Match `json:"match,omitempty"`
}

// Match defines the desired match of FerryPolicyRule
type Match struct {
	Labels    map[string]string `json:"labels,omitempty"`
	Namespace string            `json:"mamespace,omitempty"`
}

// FerryPolicyStatus defines the observed state of FerryPolicy
type FerryPolicyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// FerryPolicy is the Schema for the FerryPolicys API
type FerryPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FerryPolicySpec   `json:"spec,omitempty"`
	Status FerryPolicyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FerryPolicyList contains a list of FerryPolicy
type FerryPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FerryPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FerryPolicy{}, &FerryPolicyList{})
}
