// 🌩 Go++ 🌩 #32 · Kubernetes Operators
package widget

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	goplusplusv1 "github.com/semirm-dev/goplusplus/examples/g1-operators/api/v1"
)

// Reconciler merges a Widget's fragments into a ConfigMap it owns.
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=goplusplus.semirmahovkic.xyz,resources=widgets,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var w goplusplusv1.Widget

	// get current CR state (widget)
	if err := r.Get(ctx, req.NamespacedName, &w); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// write your logic
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: w.Name, Namespace: w.Namespace}}

	// create if missing, or update if changed, whatever your needs are
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Data = map[string]string{"config": strings.Join(w.Spec.Fragments, "\n")}
		return controllerutil.SetControllerReference(&w, cm, r.Scheme)
	})

	// TODO: handle the error properly

	return ctrl.Result{}, err
}

// SetupWithManager registers the controller and what it watches.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&goplusplusv1.Widget{}). // same for builtin resources like Pod, Node, etc.
		Owns(&corev1.ConfigMap{}).
		Named("widget").
		Complete(r)
}
