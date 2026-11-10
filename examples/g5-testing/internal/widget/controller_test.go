// 🌩 Go++ 🌩 #36 · Kubernetes Operators: Testing
package widget

import (
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	goplusplusv1 "github.com/semirm-dev/goplusplus/examples/g5-testing/api/v1"
)

const merged = "vpn: on\nfirewall: strict"

func newWidget(name string) *goplusplusv1.Widget {
	return &goplusplusv1.Widget{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec:       goplusplusv1.WidgetSpec{Fragments: []string{"vpn: on", "firewall: strict"}},
	}
}

// Fake client: call Reconcile directly. No API server, no cache, no watches.
func TestReconcile_FakeClient(t *testing.T) {
	w := newWidget("fake")
	c := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(w).Build()
	r := &Reconciler{Client: c, Scheme: scheme.Scheme}
	key := client.ObjectKeyFromObject(w)

	if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key}); err != nil {
		t.Fatal(err)
	}

	var cm corev1.ConfigMap
	if err := c.Get(ctx, key, &cm); err != nil {
		t.Fatal(err)
	}
	if cm.Data["config"] != merged {
		t.Fatalf("got %q, want %q", cm.Data["config"], merged)
	}
}

// envtest: the manager runs the controller async, so never assert once.
// Eventually polls until it passes or times out.
func TestReconcile_Envtest(t *testing.T) {
	g := NewWithT(t)
	w := newWidget("envtest")
	g.Expect(k8sClient.Create(ctx, w)).To(Succeed())

	g.Eventually(func(g Gomega) {
		var cm corev1.ConfigMap
		g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(w), &cm)).To(Succeed())
		g.Expect(cm.Data["config"]).To(Equal(merged))
	}).Should(Succeed())
}

// Only a real API server validates against the CRD. The fake client would accept it.
func TestCRDValidation_Envtest(t *testing.T) {
	w := newWidget("empty")
	w.Spec.Fragments = nil
	if err := k8sClient.Create(ctx, w); !apierrors.IsInvalid(err) {
		t.Fatalf("want invalid, got %v", err)
	}
}

// envtest runs no garbage collector, so the ConfigMap outlives its Widget.
// In a cluster, it would be deleted with its owner.
func TestNoGarbageCollector_Envtest(t *testing.T) {
	g := NewWithT(t)
	w := newWidget("delete")
	key := client.ObjectKeyFromObject(w)
	g.Expect(k8sClient.Create(ctx, w)).To(Succeed())
	g.Eventually(func() error { return k8sClient.Get(ctx, key, &corev1.ConfigMap{}) }).Should(Succeed())

	g.Expect(k8sClient.Delete(ctx, w)).To(Succeed())
	g.Consistently(func() error { return k8sClient.Get(ctx, key, &corev1.ConfigMap{}) }).Should(Succeed())
}
