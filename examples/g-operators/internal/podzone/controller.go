// 🌩 Go++ 🌩 #32 · Kubernetes Operators
package podzone

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconciler is a controller with no custom resource: it copies the zone of
// the node each Pod landed on onto the Pod's labels. The desired state lives
// on objects Kubernetes already has, so there is no CRD to install.
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;patch
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var pod corev1.Pod
	if err := r.Get(ctx, req.NamespacedName, &pod); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if pod.Spec.NodeName == "" {
		return ctrl.Result{}, nil // not scheduled yet: the update that binds it brings us back
	}

	var node corev1.Node
	if err := r.Get(ctx, client.ObjectKey{Name: pod.Spec.NodeName}, &node); err != nil {
		return ctrl.Result{}, err
	}

	zone := node.Labels[corev1.LabelTopologyZone]
	if zone == "" || pod.Labels[corev1.LabelTopologyZone] == zone {
		return ctrl.Result{}, nil
	}

	// A merge patch merges maps, so this touches one label and no others.
	// Not: r.Update. It sends the whole Pod, and the kubelet writes it too.
	patch := client.MergeFrom(pod.DeepCopy())
	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	pod.Labels[corev1.LabelTopologyZone] = zone

	return ctrl.Result{}, r.Patch(ctx, &pod, patch)
}

// SetupWithManager registers the controller and what it watches.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Named("podzone").
		Complete(r)
}
