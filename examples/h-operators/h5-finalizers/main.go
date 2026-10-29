// 🌩 Go++ 🌩 · Kubernetes Operators: Finalizers :: WIP
//
// Owner references clean up what Kubernetes owns. A finalizer is for what it
// does not: here, an AWS Shield protection, faked as a map.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/semirm-dev/goplusplus/examples/h-operators/api"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Domain-qualified, or the API server warns on every write.
const finalizer = "goplusplus.dev/shield-protection"

// Shield stands in for the AWS API: state Kubernetes cannot see or delete.
type Shield map[string]bool

func (s Shield) Protect(name string) { s[name] = true }

// Unprotect is idempotent: already gone counts as done. The controller can
// crash between this call and removing the finalizer, and will call it again.
func (s Shield) Unprotect(name string) error { delete(s, name); return nil }

type WidgetReconciler struct {
	client.Client
	Shield Shield
}

func (r *WidgetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var w api.Widget
	if err := r.Get(ctx, req.NamespacedName, &w); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Being deleted: clean up outside, then let go.
	if !w.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&w, finalizer) {
			return ctrl.Result{}, nil
		}
		if err := r.Shield.Unprotect(w.Name); err != nil {
			return ctrl.Result{}, err // the object stays Terminating until this succeeds
		}
		controllerutil.RemoveFinalizer(&w, finalizer)
		return ctrl.Result{}, r.Update(ctx, &w)
	}

	// Finalizer first, then the external resource. The other order leaks a
	// protection if the object is deleted between the two.
	if controllerutil.AddFinalizer(&w, finalizer) {
		if err := r.Update(ctx, &w); err != nil {
			return ctrl.Result{}, err
		}
	}
	r.Shield.Protect(w.Name)
	return ctrl.Result{}, nil
}

func main() {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	_ = api.AddToScheme(scheme)

	w := &api.Widget{ObjectMeta: metav1.ObjectMeta{Name: "edge", Namespace: "default"}}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(w).Build()
	r := &WidgetReconciler{Client: c, Shield: Shield{}}
	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(w)}

	reconcile := func() {
		if _, err := r.Reconcile(ctx, req); err != nil {
			log.Fatal(err)
		}
	}

	reconcile()
	fmt.Println("protected:", r.Shield["edge"])

	// 1. Delete only marks it: deletionTimestamp is set, the object stays.
	_ = c.Delete(ctx, w)
	var got api.Widget
	_ = c.Get(ctx, req.NamespacedName, &got)
	fmt.Println("after delete: terminating =", !got.DeletionTimestamp.IsZero(), "finalizers =", got.Finalizers)

	// 2. The controller cleans up and removes its finalizer; then it is gone.
	reconcile()
	err := c.Get(ctx, req.NamespacedName, &got)
	fmt.Println("protected:", r.Shield["edge"], "| widget gone:", apierrors.IsNotFound(err))
}
