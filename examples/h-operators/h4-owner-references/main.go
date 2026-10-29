// 🌩 Go++ 🌩 · Kubernetes Operators: Owner references :: WIP
//
// A controller reference on the child does two jobs: the garbage collector
// deletes the child with its parent, and Owns() maps a change to the child
// back to a reconcile of the parent.
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
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type WidgetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *WidgetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var w api.Widget
	if err := r.Get(ctx, req.NamespacedName, &w); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: w.Name, Namespace: w.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Data = map[string]string{"config": strings.Join(w.Spec.Fragments, "\n")}
		// Inside the mutate func, so a ref someone stripped is put back too.
		return controllerutil.SetControllerReference(&w, cm, r.Scheme)
	})
	return ctrl.Result{}, err
}

func (r *WidgetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.Widget{}).
		Owns(&corev1.ConfigMap{}). // someone edits the ConfigMap -> reconcile its Widget
		Complete(r)
}

func main() {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = api.AddToScheme(scheme)

	w := &api.Widget{
		ObjectMeta: metav1.ObjectMeta{Name: "edge", Namespace: "default", UID: "w-1"},
		Spec:       api.WidgetSpec{Fragments: []string{"vpn: on"}},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(w).Build()
	r := &WidgetReconciler{Client: c, Scheme: scheme}
	key := client.ObjectKeyFromObject(w)
	if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key}); err != nil {
		log.Fatal(err)
	}

	var cm corev1.ConfigMap
	_ = c.Get(ctx, key, &cm)
	ref := metav1.GetControllerOf(&cm)
	fmt.Printf("ownerRef: %s/%s controller=%t blockOwnerDeletion=%t\n",
		ref.Kind, ref.Name, *ref.Controller, *ref.BlockOwnerDeletion)

	// What Owns() does with an event on the ConfigMap: follow the controller
	// ref up to the parent's key. The owner is always in the child's namespace.
	parent := types.NamespacedName{Namespace: cm.Namespace, Name: ref.Name}
	fmt.Println("ConfigMap changed -> reconcile", parent)

	// Only one controller per object.
	other := &api.Widget{ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "default", UID: "w-2"}}
	err := controllerutil.SetControllerReference(other, &cm, scheme)
	fmt.Println("second controller:", err)

	// Note: neither the fake client nor envtest runs the garbage collector,
	// so deleting the Widget here would leave the ConfigMap behind. That
	// cleanup is kube-controller-manager's job, in a real cluster.
}
