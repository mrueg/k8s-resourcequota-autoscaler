/*
Copyright 2021 Manuel Rüger <manuel@rueg.eu>.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ManagedResourceQuotaSpec defines the desired state of ManagedResourceQuota
type ManagedResourceQuotaSpec struct {
	Template ResourceQuotaTemplateSpec `json:"template,omitempty"`
}

type ResourceQuotaTemplateSpec struct {
	// hard is the set of desired hard limits for each named resource.
	// More info: https://kubernetes.io/docs/concepts/policy/resource-quotas/
	// +optional
	Hard ResourceList `json:"hard,omitempty"`
	// A collection of filters that must match each object tracked by a quota.
	// If not specified, the quota matches all objects.
	// +optional
	Scopes []corev1.ResourceQuotaScope `json:"scopes,omitempty"`
	// scopeSelector is also a collection of filters like scopes that must match each object tracked by a quota
	// but expressed using ScopeSelectorOperator in combination with possible values.
	// For a resource to match, both scopes AND scopeSelector (if specified in spec), must be matched.
	// +optional
	ScopeSelector *corev1.ScopeSelector `json:"scopeSelector,omitempty"`
}

type ResourceList map[corev1.ResourceName]string

// ManagedResourceQuotaStatus defines the observed state of ManagedResourceQuota
type ManagedResourceQuotaStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ManagedResourceQuota is the Schema for the managedresourcequota API
type ManagedResourceQuota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedResourceQuotaSpec   `json:"spec,omitempty"`
	Status ManagedResourceQuotaStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ManagedResourceQuotaList contains a list of ManagedResourceQuota
type ManagedResourceQuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedResourceQuota `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ManagedResourceQuota{}, &ManagedResourceQuotaList{})
}
