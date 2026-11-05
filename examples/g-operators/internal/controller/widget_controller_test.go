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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	goplusplusv1 "github.com/semirm-dev/goplusplus/examples/g-operators/api/v1"
)

// These run against envtest: a real kube-apiserver and etcd, and the manager
// started in suite_test.go. The controller is asynchronous, so every check on
// its work is Eventually, never a single assertion.
var _ = Describe("Widget Controller", func() {
	const merged = "vpn: on\nfirewall: strict"

	newWidget := func(name string) *goplusplusv1.Widget {
		return &goplusplusv1.Widget{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace},
			Spec:       goplusplusv1.WidgetSpec{Fragments: []string{vpnOn, "firewall: strict"}},
		}
	}

	It("merges the fragments into a ConfigMap it owns, and reports which spec is Available", func() {
		w := newWidget("merge")
		Expect(k8sClient.Create(ctx, w)).To(Succeed())
		key := client.ObjectKeyFromObject(w)

		Eventually(func(g Gomega) {
			var cm corev1.ConfigMap
			g.Expect(k8sClient.Get(ctx, key, &cm)).To(Succeed())
			g.Expect(cm.Data["config"]).To(Equal(merged))
			g.Expect(metav1.GetControllerOf(&cm).Name).To(Equal(w.Name))
		}).Should(Succeed())

		Eventually(func(g Gomega) {
			var got goplusplusv1.Widget
			g.Expect(k8sClient.Get(ctx, key, &got)).To(Succeed())
			g.Expect(got.Status.ObservedGeneration).To(Equal(got.Generation))
			g.Expect(meta.IsStatusConditionTrue(got.Status.Conditions, "Available")).To(BeTrue())
			g.Expect(got.Finalizers).To(ContainElement(shieldFinalizer))
		}).Should(Succeed())
		Expect(shield.has(key.String())).To(BeTrue())
	})

	It("rejects a Widget with no fragments: the CRD schema is enforced by the API server", func() {
		w := newWidget("empty")
		w.Spec.Fragments = nil
		Expect(errors.IsInvalid(k8sClient.Create(ctx, w))).To(BeTrue())
	})

	It("puts a hand-edited ConfigMap back: Owns maps the change to its Widget", func() {
		w := newWidget("drift")
		Expect(k8sClient.Create(ctx, w)).To(Succeed())
		key := client.ObjectKeyFromObject(w)

		var cm corev1.ConfigMap
		Eventually(func() error { return k8sClient.Get(ctx, key, &cm) }).Should(Succeed())
		cm.Data["config"] = "tampered"
		Expect(k8sClient.Update(ctx, &cm)).To(Succeed())

		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, key, &cm)).To(Succeed())
			g.Expect(cm.Data["config"]).To(Equal(merged))
		}).Should(Succeed())
	})

	It("ignores status in a plain Update once the status subresource is on", func() {
		w := newWidget("status")
		Expect(k8sClient.Create(ctx, w)).To(Succeed())

		// The controller writes this Widget too, so a 409 here is expected: retry it.
		var got goplusplusv1.Widget
		Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(w), &got); err != nil {
				return err
			}
			got.Status.ObservedGeneration = 99
			return k8sClient.Update(ctx, &got)
		})).To(Succeed())
		Expect(got.Status.ObservedGeneration).NotTo(Equal(int64(99))) // the server's answer, decoded back
	})

	It("unprotects on delete, and leaves the ConfigMap: envtest runs no garbage collector", func() {
		w := newWidget("delete")
		Expect(k8sClient.Create(ctx, w)).To(Succeed())
		key := client.ObjectKeyFromObject(w)
		Eventually(func() bool { return shield.has(key.String()) }).Should(BeTrue())

		Expect(k8sClient.Delete(ctx, w)).To(Succeed())
		Eventually(func() bool {
			return errors.IsNotFound(k8sClient.Get(ctx, key, &goplusplusv1.Widget{}))
		}).Should(BeTrue())
		Expect(shield.has(key.String())).To(BeFalse())

		// In a cluster, kube-controller-manager would delete this with its owner.
		Consistently(func() error {
			return k8sClient.Get(ctx, key, &corev1.ConfigMap{})
		}).Should(Succeed())
	})

	It("collapses repeated events for one key into one queued reconcile", func() {
		q := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[ctrl.Request]())
		defer q.ShutDown()

		req := ctrl.Request{NamespacedName: client.ObjectKey{Namespace: testNamespace, Name: "edge"}}
		for range 3 {
			q.Add(req)
		}
		Expect(q.Len()).To(Equal(1))
	})
})
