/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	goplusplusv1 "github.com/semirm-dev/goplusplus/examples/g-operators/api/v1"
)

// Every object a controller reads can already be out of date: in a manager,
// Get reads the informer cache and only writes reach the API server. These
// run on the fake client, where nothing else writes in between.
var _ = Describe("Stale reads", func() {
	var (
		c      client.Client
		key    client.ObjectKey
		a, b   goplusplusv1.Widget
		before *goplusplusv1.Widget // b as it was read
	)

	BeforeEach(func() {
		w := &goplusplusv1.Widget{
			ObjectMeta: metav1.ObjectMeta{Name: "edge", Namespace: testNamespace},
			Spec:       goplusplusv1.WidgetSpec{Fragments: []string{vpnOn}},
		}
		c = fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(w).Build()
		key = client.ObjectKeyFromObject(w)

		// Two readers, one version; a writes first.
		Expect(c.Get(ctx, key, &a)).To(Succeed())
		Expect(c.Get(ctx, key, &b)).To(Succeed())
		before = b.DeepCopy()

		a.Spec.Fragments = append(a.Spec.Fragments, "firewall: strict")
		Expect(c.Update(ctx, &a)).To(Succeed())
		b.Spec.Fragments = append(b.Spec.Fragments, "ids: on")
	})

	It("Update sends the resourceVersion it read, so a stale one is a 409", func() {
		Expect(apierrors.IsConflict(c.Update(ctx, &b))).To(BeTrue())
	})

	It("a merge patch carries no resourceVersion, wins, and replaces the whole list", func() {
		Expect(c.Patch(ctx, &b, client.MergeFrom(before))).To(Succeed())

		var now goplusplusv1.Widget
		Expect(c.Get(ctx, key, &now)).To(Succeed())
		Expect(now.Spec.Fragments).To(Equal([]string{vpnOn, "ids: on"})) // a's fragment is gone
	})

	It("the optimistic lock puts the resourceVersion, and the conflict, back", func() {
		err := c.Patch(ctx, &b, client.MergeFromWithOptions(before, client.MergeFromWithOptimisticLock{}))
		Expect(apierrors.IsConflict(err)).To(BeTrue())
	})

	It("a conflict is fixed by reading again and redoing the change", func() {
		Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
			var fresh goplusplusv1.Widget
			if err := c.Get(ctx, key, &fresh); err != nil {
				return err
			}
			fresh.Spec.Fragments = append(fresh.Spec.Fragments, "ids: on")
			return c.Update(ctx, &fresh)
		})).To(Succeed())

		var now goplusplusv1.Widget
		Expect(c.Get(ctx, key, &now)).To(Succeed())
		Expect(now.Spec.Fragments).To(Equal([]string{vpnOn, "firewall: strict", "ids: on"}))
	})
})
