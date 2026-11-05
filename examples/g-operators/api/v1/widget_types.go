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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// WidgetSpec is written by the user: the config fragments to merge, in order.
type WidgetSpec struct {
	// fragments are merged, one per line, into the Widget's ConfigMap.
	// +kubebuilder:validation:MinItems=1
	// +required
	Fragments []string `json:"fragments"`
}

// WidgetStatus is written by the controller, and only by the controller.
type WidgetStatus struct {
	// observedGeneration is the metadata.generation this status describes.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// conditions represent the current state of the Widget resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Widget is the Schema for the widgets API
type Widget struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of Widget
	// +required
	Spec WidgetSpec `json:"spec"`

	// status defines the observed state of Widget
	// +optional
	Status WidgetStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// WidgetList contains a list of Widget
type WidgetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Widget `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &Widget{}, &WidgetList{})
		return nil
	})
}
