// 🌩 Go++ 🌩 · Kubernetes Operators: Spec vs status :: WIP
//
// The user writes spec, the controller writes status, and observedGeneration
// says which spec the status is describing.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/semirm-dev/goplusplus/examples/h-operators/api"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type WidgetReconciler struct {
	client.Client
}

func (r *WidgetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var w api.Widget
	if err := r.Get(ctx, req.NamespacedName, &w); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// ... converge the ConfigMap here, as in H1 ...

	// SetStatusCondition only moves LastTransitionTime when Status flips,
	// so "Ready since" stays true across every pass that changes nothing.
	meta.SetStatusCondition(&w.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "ConfigMerged",
		Message:            fmt.Sprintf("%d fragments merged", len(w.Spec.Fragments)),
		ObservedGeneration: w.Generation,
	})
	w.Status.ObservedGeneration = w.Generation

	// Status goes through its own subresource. r.Update would drop it.
	return ctrl.Result{}, r.Status().Update(ctx, &w)
}

func main() {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	_ = api.AddToScheme(scheme)

	w := &api.Widget{
		// The API server bumps generation on every spec change. The fake
		// client does not, so it is set by hand here.
		ObjectMeta: metav1.ObjectMeta{Name: "edge", Namespace: "default", Generation: 3},
		Spec:       api.WidgetSpec{Fragments: []string{"vpn: on", "firewall: strict"}},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(w).
		WithStatusSubresource(&api.Widget{}). // same as `subresources: status: {}` in the CRD
		Build()
	key := client.ObjectKeyFromObject(w)

	// 1. A plain Update cannot write status once the subresource is on.
	var got api.Widget
	_ = c.Get(ctx, key, &got)
	got.Status.ObservedGeneration = 99
	_ = c.Update(ctx, &got)
	_ = c.Get(ctx, key, &got)
	fmt.Println("status after Update:", got.Status.ObservedGeneration)

	// 2. The controller writes it through Status().
	r := &WidgetReconciler{Client: c}
	if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key}); err != nil {
		log.Fatal(err)
	}
	_ = c.Get(ctx, key, &got)
	ready := meta.FindStatusCondition(got.Status.Conditions, "Ready")
	fmt.Printf("generation %d, observedGeneration %d, Ready=%s (%s)\n",
		got.Generation, got.Status.ObservedGeneration, ready.Status, ready.Message)
}
