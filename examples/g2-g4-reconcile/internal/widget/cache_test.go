// 🌩 Go++ 🌩 #33 · Kubernetes Operators: Reconcile loop
package widget

import (
	"context"
	"slices"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	goplusplusv1 "github.com/semirm-dev/goplusplus/examples/g2-g4-reconcile/api/v1"
)

// Get reads the informer cache, so the object can already be out of date.
// Update sends the resourceVersion it read, and a stale one is a 409.
func TestStaleUpdateConflicts(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	if err := goplusplusv1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	w := &goplusplusv1.Widget{
		ObjectMeta: metav1.ObjectMeta{Name: "edge", Namespace: "default"},
		Spec:       goplusplusv1.WidgetSpec{Fragments: []string{"vpn: on"}},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(w).Build()
	key := client.ObjectKeyFromObject(w)

	// Two readers, one version; a writes first.
	var a, b goplusplusv1.Widget
	if err := c.Get(ctx, key, &a); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(ctx, key, &b); err != nil {
		t.Fatal(err)
	}
	a.Spec.Fragments = append(a.Spec.Fragments, "firewall: strict")
	if err := c.Update(ctx, &a); err != nil {
		t.Fatal(err)
	}

	b.Spec.Fragments = append(b.Spec.Fragments, "ids: on")
	if err := c.Update(ctx, &b); !apierrors.IsConflict(err) {
		t.Fatalf("want 409, got %v", err)
	}

	// The fix: read again and redo the change. In Reconcile, returning the
	// error does the same, the requeue reads it fresh.
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var fresh goplusplusv1.Widget
		if err := c.Get(ctx, key, &fresh); err != nil {
			return err
		}
		fresh.Spec.Fragments = append(fresh.Spec.Fragments, "ids: on")
		return c.Update(ctx, &fresh)
	})
	if err != nil {
		t.Fatal(err)
	}

	var now goplusplusv1.Widget
	if err := c.Get(ctx, key, &now); err != nil {
		t.Fatal(err)
	}
	if want := []string{"vpn: on", "firewall: strict", "ids: on"}; !slices.Equal(now.Spec.Fragments, want) {
		t.Fatalf("got %v, want %v", now.Spec.Fragments, want)
	}
}
