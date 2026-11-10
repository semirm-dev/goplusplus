// 🌩 Go++ 🌩 #33-#35 · Kubernetes Operators: Reconcile loop, Spec and status, Cleanup
package widget

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	goplusplusv1 "github.com/semirm-dev/goplusplus/examples/g2-g4-reconcile/api/v1"
)

const fin = "goplusplus.semirmahovkic.xyz/shield"

// Reconciler merges a Widget's fragments into a ConfigMap it owns, and keeps
// the Widget protected by AWS Shield while it exists.
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Shield Shield
}

// +kubebuilder:rbac:groups=goplusplus.semirmahovkic.xyz,resources=widgets,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=goplusplus.semirmahovkic.xyz,resources=widgets/status,verbs=update
// +kubebuilder:rbac:groups=goplusplus.semirmahovkic.xyz,resources=widgets/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// G2: req is only namespace/name, no event. Read the current state.
	var w goplusplusv1.Widget
	if err := r.Get(ctx, req.NamespacedName, &w); err != nil {
		// Not: return err, a deleted Widget would be retried forever.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	name := req.String()

	// G4: being deleted. Clean up outside first, then remove the finalizer.
	if !w.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&w, fin) {
			return ctrl.Result{}, nil
		}
		if err := r.Shield.Unprotect(ctx, name); err != nil {
			return ctrl.Result{}, err // stays Terminating until this works
		}
		controllerutil.RemoveFinalizer(&w, fin)
		return ctrl.Result{}, r.Update(ctx, &w)
	}

	// G4: finalizer first, then the external resource.
	if controllerutil.AddFinalizer(&w, fin) {
		if err := r.Update(ctx, &w); err != nil {
			return ctrl.Result{}, err // a 409 too: the requeue reads it fresh
		}
	}
	if err := r.Shield.Protect(ctx, name); err != nil {
		return ctrl.Result{}, err
	}

	// G4: the owner reference. Delete the Widget and Kubernetes deletes the ConfigMap.
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: w.Name, Namespace: w.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Data = map[string]string{"config": strings.Join(w.Spec.Fragments, "\n")}
		return controllerutil.SetControllerReference(&w, cm, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	// G3: report what was done.
	meta.SetStatusCondition(&w.Status.Conditions, metav1.Condition{
		Type:               "Available",
		Status:             metav1.ConditionTrue,
		Reason:             "ConfigMerged",
		Message:            fmt.Sprintf("%d fragments merged", len(w.Spec.Fragments)),
		ObservedGeneration: w.Generation,
	})
	w.Status.ObservedGeneration = w.Generation
	// Not: r.Update, with the status subresource on it drops status.
	if err := r.Status().Update(ctx, &w); err != nil {
		return ctrl.Result{}, err
	}

	// G2: nothing watches AWS for us, so look again later.
	return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil
}

// SetupWithManager registers the controller and what it watches.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&goplusplusv1.Widget{}).
		Owns(&corev1.ConfigMap{}). // someone edits the ConfigMap -> reconcile its Widget
		Named("widget").
		Complete(r)
}
