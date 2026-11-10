// 🌩 Go++ 🌩 #33-#35 · Kubernetes Operators
package main

import (
	"log"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	goplusplusv1 "github.com/semirm-dev/goplusplus/examples/g2-g4-reconcile/api/v1"
	"github.com/semirm-dev/goplusplus/examples/g2-g4-reconcile/internal/widget"
)

func main() {
	ctrl.SetLogger(zap.New())

	// The scheme tells the client which Go type is which kind: the built-in ones and our Widget.
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(goplusplusv1.AddToScheme(scheme))

	// TODO: metrics, health probes and leader election before running it in production.
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{Scheme: scheme})
	if err != nil {
		log.Fatal(err)
	}

	if err := (&widget.Reconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()}).SetupWithManager(mgr); err != nil {
		log.Fatal(err)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Fatal(err)
	}
}
