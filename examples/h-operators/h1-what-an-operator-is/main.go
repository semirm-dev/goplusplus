// 🌩 Go++ 🌩 · Kubernetes Operators: What an Operator is :: WIP
//
// An Operator is two things: a custom resource that says what you want, and a
// controller that makes the cluster match it. The controller half also works
// on its own, over kinds Kubernetes already has: both are below. The fake
// client stands in for the API server, so this runs without a cluster.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/semirm-dev/goplusplus/examples/h-operators/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type WidgetReconciler struct {
	client.Client
}

// Reconcile turns desired state (the Widget) into actual state (a ConfigMap).
func (r *WidgetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var w api.Widget
	if err := r.Get(ctx, req.NamespacedName, &w); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err) // gone: nothing left to converge
	}

	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: w.Name, Namespace: w.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Data = map[string]string{"config": strings.Join(w.Spec.Fragments, "\n")}
		return nil
	})
	return ctrl.Result{}, err
}

// In a real binary the manager owns the watch and calls Reconcile for you.
func (r *WidgetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&api.Widget{}).Complete(r)
}

// ZoneReconciler is a controller with no custom resource: it reconciles Pods,
// copying the zone of the node each one landed on onto the Pod's labels.
// The desired state lives on objects that already exist.
type ZoneReconciler struct {
	client.Client
}

func (r *ZoneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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
	if pod.Labels[corev1.LabelTopologyZone] == zone {
		return ctrl.Result{}, nil
	}

	patch := client.MergeFrom(pod.DeepCopy())
	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	pod.Labels[corev1.LabelTopologyZone] = zone
	return ctrl.Result{}, r.Patch(ctx, &pod, patch)
}

// The same builder, a built-in kind, and no CRD to install.
func (r *ZoneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&corev1.Pod{}).Complete(r)
}

func main() {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = api.AddToScheme(scheme)

	// 1. The user writes desired state.
	w := &api.Widget{
		ObjectMeta: metav1.ObjectMeta{Name: "edge", Namespace: "default"},
		Spec:       api.WidgetSpec{Fragments: []string{"vpn: on", "firewall: strict"}},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(w).Build()

	// 2. The controller makes it true.
	r := &WidgetReconciler{Client: c}
	key := client.ObjectKeyFromObject(w)
	if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key}); err != nil {
		log.Fatal(err)
	}

	var cm corev1.ConfigMap
	if err := c.Get(ctx, key, &cm); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("ConfigMap %s:\n%s\n", key, cm.Data["config"])

	// 3. No custom resource: a Pod scheduled on a node in eu-west-1a.
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{
		Name:   "node-a",
		Labels: map[string]string{corev1.LabelTopologyZone: "eu-west-1a"},
	}}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "api-0", Namespace: "default"},
		Spec:       corev1.PodSpec{NodeName: "node-a"},
	}
	_ = c.Create(ctx, node)
	_ = c.Create(ctx, pod)

	zr := &ZoneReconciler{Client: c}
	podKey := client.ObjectKeyFromObject(pod)
	if _, err := zr.Reconcile(ctx, ctrl.Request{NamespacedName: podKey}); err != nil {
		log.Fatal(err)
	}
	_ = c.Get(ctx, podKey, pod)
	fmt.Printf("Pod %s zone: %s\n", podKey, pod.Labels[corev1.LabelTopologyZone])
}
