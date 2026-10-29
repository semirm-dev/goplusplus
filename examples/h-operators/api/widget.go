// Package api is the one custom resource every H example reconciles.
//
// A Widget lists config fragments; its controller merges them into a single
// ConfigMap. The same shape as the SFCN config Operator, at toy size.
package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var GroupVersion = schema.GroupVersion{Group: "goplusplus.dev", Version: "v1"}

// WidgetSpec is written by the user.
type WidgetSpec struct {
	Fragments []string `json:"fragments,omitempty"`
}

// WidgetStatus is written by the controller, and only by the controller.
type WidgetStatus struct {
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

type Widget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WidgetSpec   `json:"spec,omitempty"`
	Status WidgetStatus `json:"status,omitempty"`
}

type WidgetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Widget `json:"items"`
}

// AddToScheme registers Widget so clients can map the Go type to its API path.
func AddToScheme(s *runtime.Scheme) error {
	s.AddKnownTypes(GroupVersion, &Widget{}, &WidgetList{})
	metav1.AddToGroupVersion(s, GroupVersion)
	return nil
}

// DeepCopyObject is what controller-gen would generate. Written by hand here
// because there is exactly one type: every slice is copied, never shared,
// since the cache hands out the same object to every reader.
func (w *Widget) DeepCopyObject() runtime.Object {
	out := *w
	w.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec.Fragments = append([]string(nil), w.Spec.Fragments...)
	out.Status.Conditions = nil
	for _, c := range w.Status.Conditions {
		out.Status.Conditions = append(out.Status.Conditions, *c.DeepCopy())
	}
	return &out
}

func (l *WidgetList) DeepCopyObject() runtime.Object {
	out := *l
	l.ListMeta.DeepCopyInto(&out.ListMeta)
	out.Items = make([]Widget, len(l.Items))
	for i := range l.Items {
		out.Items[i] = *l.Items[i].DeepCopyObject().(*Widget)
	}
	return &out
}
