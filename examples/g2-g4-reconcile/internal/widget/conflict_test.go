// 🌩 Go++ 🌩 #33 · Kubernetes Operators: Reconcile loop
package widget

import (
	"context"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	goplusplusv1 "github.com/semirm-dev/goplusplus/examples/g2-g4-reconcile/api/v1"
)

// Update sends the resourceVersion it read, so updating a stale object is a 409.
// In Reconcile just return the error, the next run reads it fresh.
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

	// a and b read the same version, a updates first
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

	// b is stale now
	b.Spec.Fragments = append(b.Spec.Fragments, "ids: on")
	if err := c.Update(ctx, &b); !apierrors.IsConflict(err) {
		t.Fatalf("want 409, got %v", err)
	}
}
