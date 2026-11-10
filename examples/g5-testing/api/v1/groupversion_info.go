// Package v1 is the Widget API.
// +kubebuilder:object:generate=true
// +groupName=goplusplus.semirmahovkic.xyz
package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	GroupVersion  = schema.GroupVersion{Group: "goplusplus.semirmahovkic.xyz", Version: "v1"}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme
)
