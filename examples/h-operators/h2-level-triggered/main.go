// 🌩 Go++ 🌩 · Kubernetes Operators: Reconcile is level-triggered :: WIP
//
// Reconcile gets a key, not an event. It reads the world fresh every time, so
// it can run once or fifty times and land in the same place.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/semirm-dev/goplusplus/examples/h-operators/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type WidgetReconciler struct {
	client.Client
}

func (r *WidgetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// req is just namespace/name. Which event fired, and how many, is not passed in.
	var w api.Widget
	if err := r.Get(ctx, req.NamespacedName, &w); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: w.Name, Namespace: w.Namespace}}
	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Data = map[string]string{"config": strings.Join(w.Spec.Fragments, "\n")}
		return nil
	})
	if err != nil {
		return ctrl.Result{}, err // requeued with per-key exponential backoff
	}
	fmt.Println("  configmap:", op)

	// Nothing watches the outside world for us, so we poll it on a timer
	// instead of remembering to come back.
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func main() {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = api.AddToScheme(scheme)

	w := &api.Widget{
		ObjectMeta: metav1.ObjectMeta{Name: "edge", Namespace: "default"},
		Spec:       api.WidgetSpec{Fragments: []string{"vpn: on"}},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(w).Build()
	r := &WidgetReconciler{Client: c}
	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(w)}

	// 1. Three events for one key collapse into one item on the queue.
	q := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[ctrl.Request]())
	defer q.ShutDown()
	for range 3 {
		q.Add(req)
	}
	fmt.Println("events: 3, queued:", q.Len())

	// 2. Running it again changes nothing: it compares, it does not replay.
	for i := range 3 {
		fmt.Println("reconcile", i+1)
		if _, err := r.Reconcile(ctx, req); err != nil {
			log.Fatal(err)
		}
	}

	// 3. Someone deletes the ConfigMap by hand. The next pass puts it back,
	// because it reads what is, not what happened.
	_ = c.Delete(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "edge", Namespace: "default"}})
	fmt.Println("after drift")
	if _, err := r.Reconcile(ctx, req); err != nil {
		log.Fatal(err)
	}
}
