// 🌩 Go++ 🌩 · Kubernetes Operators: Testing with envtest :: WIP
//
// envtest starts a real kube-apiserver and etcd, and nothing else. No
// kube-controller-manager means no garbage collector: the last check here
// passes only because envtest is not a cluster.
//
//	go run sigs.k8s.io/controller-runtime/tools/setup-envtest@latest use -p path
//	KUBEBUILDER_ASSETS=<that path> go test -race ./examples/h-operators/h7-envtest
package h7

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/semirm-dev/goplusplus/examples/h-operators/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
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
		return controllerutil.SetControllerReference(&w, cm, r.Scheme)
	})
	return ctrl.Result{}, err
}

func TestWidget(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("KUBEBUILDER_ASSETS not set: run setup-envtest first")
	}

	env := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd")},
		ErrorIfCRDPathMissing: true,
	}
	cfg, err := env.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = env.Stop() })

	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = api.AddToScheme(scheme)

	// The real manager: informer cache, workqueue, the lot.
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme, Metrics: metricsserver.Options{BindAddress: "0"}})
	if err != nil {
		t.Fatal(err)
	}
	err = ctrl.NewControllerManagedBy(mgr).
		For(&api.Widget{}).
		Owns(&corev1.ConfigMap{}).
		Complete(&WidgetReconciler{Client: mgr.GetClient(), Scheme: scheme})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = mgr.Start(ctx) }()

	// The test talks to the API server directly, not through the cache.
	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatal(err)
	}

	w := &api.Widget{
		ObjectMeta: metav1.ObjectMeta{Name: "edge", Namespace: "default"},
		Spec:       api.WidgetSpec{Fragments: []string{"vpn: on", "firewall: strict"}},
	}
	if err := c.Create(ctx, w); err != nil {
		t.Fatal(err)
	}
	key := client.ObjectKeyFromObject(w)

	// The controller runs asynchronously, so assert eventually, never once.
	var cm corev1.ConfigMap
	eventually(t, func() bool {
		return c.Get(ctx, key, &cm) == nil && cm.Data["config"] == "vpn: on\nfirewall: strict"
	})

	// Drift: edit the child by hand. Owns() maps it back to the Widget,
	// and the controller puts it right.
	cm.Data["config"] = "tampered"
	if err := c.Update(ctx, &cm); err != nil {
		t.Fatal(err)
	}
	eventually(t, func() bool {
		return c.Get(ctx, key, &cm) == nil && cm.Data["config"] == "vpn: on\nfirewall: strict"
	})

	// No garbage collector here: the owned ConfigMap outlives its Widget.
	if err := c.Delete(ctx, w); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	if err := c.Get(ctx, key, &cm); err != nil {
		t.Fatalf("expected the ConfigMap to survive without GC, got %v", err)
	}
}

func eventually(t *testing.T, ok func() bool) {
	t.Helper()
	for range 50 {
		if ok() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("condition never became true")
}
