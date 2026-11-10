package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// WidgetSpec is the desired state: the config fragments to merge, in order.
type WidgetSpec struct {
	// +kubebuilder:validation:MinItems=1
	Fragments []string `json:"fragments"`
}

// Widget is the custom resource. The markers generate its CRD.
// +kubebuilder:object:root=true
type Widget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec WidgetSpec `json:"spec"`
}

// WidgetList is what a List call returns.
// +kubebuilder:object:root=true
type WidgetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Widget `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Widget{}, &WidgetList{})
}
