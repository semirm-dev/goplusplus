// 🌩 Go++ 🌩 #36 · Kubernetes Operators: Testing
package widget

import (
	"context"
	"log"
	"os"
	"testing"

	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	goplusplusv1 "github.com/semirm-dev/goplusplus/examples/g5-testing/api/v1"
)

var (
	ctx       = context.Background()
	k8sClient client.Client // talks to the API server directly, no cache
)

func TestMain(m *testing.M) {
	if err := goplusplusv1.AddToScheme(scheme.Scheme); err != nil {
		log.Fatal(err)
	}

	// A real kube-apiserver and etcd, with our CRD installed.
	// Nothing else runs: no garbage collector, scheduler or kubelet.
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{"../../config/crd/bases"},
		ErrorIfCRDPathMissing: true,
	}
	cfg, err := testEnv.Start()
	if err != nil {
		log.Fatal(err)
	}

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		log.Fatal(err)
	}

	// The real manager: cache, workqueue and watches, the same as in the cluster.
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme.Scheme, Metrics: metricsserver.Options{BindAddress: "0"}})
	if err != nil {
		log.Fatal(err)
	}
	if err := (&Reconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()}).SetupWithManager(mgr); err != nil {
		log.Fatal(err)
	}
	mgrCtx, stop := context.WithCancel(ctx)
	go func() {
		if err := mgr.Start(mgrCtx); err != nil {
			log.Fatal(err)
		}
	}()

	code := m.Run()
	stop()
	_ = testEnv.Stop()
	os.Exit(code)
}
