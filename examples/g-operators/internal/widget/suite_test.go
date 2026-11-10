// 🌩 Go++ 🌩 #36 · Kubernetes Operators: Testing with envtest
package widget

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	goplusplusv1 "github.com/semirm-dev/goplusplus/examples/g-operators/api/v1"
)

const (
	testNamespace = "default"
	vpnOn         = "vpn: on"
)

var (
	ctx       context.Context
	cancel    context.CancelFunc
	testEnv   *envtest.Environment
	k8sClient client.Client
	shield    *memShield
)

func TestWidget(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Widget Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())

	Expect(goplusplusv1.AddToScheme(scheme.Scheme)).To(Succeed())

	// A real kube-apiserver and etcd, with our CRD installed. No controllers
	// run except ours: no garbage collector, no scheduler, no kubelet.
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
		BinaryAssetsDirectory: firstEnvtestBinaryDir(),
	}
	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())

	// The real manager: informer cache, workqueue and watches, the same as
	// in the cluster. Tests talk to the API server through k8sClient, which
	// skips the cache, and wait for the controller with Eventually.
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:  scheme.Scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
	})
	Expect(err).NotTo(HaveOccurred())

	shield = &memShield{protected: map[string]bool{}}
	Expect((&Reconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme(), Shield: shield}).
		SetupWithManager(mgr)).To(Succeed())

	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(ctx)).To(Succeed())
	}()
})

var _ = AfterSuite(func() {
	cancel()
	Expect(testEnv.Stop()).To(Succeed())
})

// firstEnvtestBinaryDir finds the binaries `make setup-envtest` downloaded,
// so the tests also run from an IDE without KUBEBUILDER_ASSETS set.
func firstEnvtestBinaryDir() string {
	base := filepath.Join("..", "..", "bin", "k8s")
	entries, _ := os.ReadDir(base)
	for _, e := range entries {
		if e.IsDir() {
			return filepath.Join(base, e.Name())
		}
	}
	return ""
}

// memShield records protections in memory, so a test can see what the
// controller did outside the cluster.
type memShield struct {
	mu        sync.Mutex
	protected map[string]bool
}

func (s *memShield) Protect(_ context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.protected[name] = true
	return nil
}

func (s *memShield) Unprotect(_ context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.protected, name)
	return nil
}

func (s *memShield) has(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.protected[name]
}
