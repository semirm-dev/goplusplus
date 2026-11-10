// 🌩 Go++ 🌩 #32 · Kubernetes Operators
package podzone

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconciler has no custom resource: it copies the zone of the node a Pod
// runs on onto the Pod's labels.
type Reconciler struct {
	client.Client
}

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var pod corev1.Pod
	if err := r.Get(ctx, req.NamespacedName, &pod); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if pod.Spec.NodeName == "" {
		return ctrl.Result{}, nil // not scheduled yet, scheduling it brings us back
	}

	var node corev1.Node
	if err := r.Get(ctx, client.ObjectKey{Name: pod.Spec.NodeName}, &node); err != nil {
		return ctrl.Result{}, err
	}

	zone := node.Labels[corev1.LabelTopologyZone]
	if zone == "" || pod.Labels[corev1.LabelTopologyZone] == zone {
		return ctrl.Result{}, nil
	}
	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	pod.Labels[corev1.LabelTopologyZone] = zone
	return ctrl.Result{}, r.Update(ctx, &pod)
}

// SetupWithManager registers the controller and what it watches.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Named("podzone").
		Complete(r)
}
