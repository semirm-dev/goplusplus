// 🌩 Go++ 🌩 · Kubernetes Operators: The cache and conflicts :: WIP
//
// Every object you read can already be out of date. resourceVersion is how
// the API server notices, and a plain merge patch is how you stop it noticing.
package main

import (
	"context"
	"fmt"

	"github.com/semirm-dev/goplusplus/examples/h-operators/api"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func main() {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	_ = api.AddToScheme(scheme)

	w := &api.Widget{
		ObjectMeta: metav1.ObjectMeta{Name: "edge", Namespace: "default"},
		Spec:       api.WidgetSpec{Fragments: []string{"vpn: on"}},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(w).Build()
	key := client.ObjectKeyFromObject(w)

	// Two readers, one version. In a manager, Get reads from the informer
	// cache and only writes go to the API server, so this is the normal case.
	var a, b api.Widget
	_ = c.Get(ctx, key, &a)
	_ = c.Get(ctx, key, &b)
	before := b.DeepCopyObject().(*api.Widget) // b as it was read

	a.Spec.Fragments = append(a.Spec.Fragments, "firewall: strict")
	_ = c.Update(ctx, &a)

	// 1. Update sends b's resourceVersion. It is stale, so the server says 409.
	b.Spec.Fragments = append(b.Spec.Fragments, "ids: on")
	err := c.Update(ctx, &b)
	fmt.Println("update b:", apierrors.IsConflict(err))

	// 2. The same change as a merge patch carries no resourceVersion and wins.
	// A list in a JSON merge patch is replaced whole, so a's fragment is gone.
	_ = c.Patch(ctx, &b, client.MergeFrom(before))
	var now api.Widget
	_ = c.Get(ctx, key, &now)
	fmt.Println("after merge patch:", now.Spec.Fragments)

	// 3. With the optimistic lock the patch carries resourceVersion again.
	stale := before.DeepCopyObject().(*api.Widget)
	stale.Spec.Fragments = append(stale.Spec.Fragments, "nat: on")
	err = c.Patch(ctx, stale, client.MergeFromWithOptions(before, client.MergeFromWithOptimisticLock{}))
	fmt.Println("locked patch:", apierrors.IsConflict(err))

	// 4. A conflict means read again and redo the change, not force it through.
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var fresh api.Widget
		if err := c.Get(ctx, key, &fresh); err != nil {
			return err
		}
		fresh.Spec.Fragments = append(fresh.Spec.Fragments, "nat: on")
		return c.Update(ctx, &fresh)
	})
	_ = c.Get(ctx, key, &now)
	fmt.Println("retried:", err, now.Spec.Fragments)
}
