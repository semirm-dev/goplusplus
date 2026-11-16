// 🌩 Go++ 🌩 #31-#32 · Kubernetes Operators: Reconcile loop, Spec and status
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

// RBAC: what the controller may do, controller-gen turns these into a ClusterRole.
// widgets/status and widgets/finalizers are separate, so users can edit
// a Widget without being able to touch its status.
//
// +kubebuilder:rbac:groups=goplusplus.semirmahovkic.xyz,resources=widgets,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=goplusplus.semirmahovkic.xyz,resources=widgets/status,verbs=update
// +kubebuilder:rbac:groups=goplusplus.semirmahovkic.xyz,resources=widgets/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var w goplusplusv1.Widget

	// #### Get the current object ####

	// # 1. read the current object first, req has only namespace/name, no event
	if err := r.Get(ctx, req.NamespacedName, &w); err != nil {
		// not found = already deleted, nothing to do (returning err would retry it forever)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	name := req.String()

	// # 2. G3: being deleted? clean up, remove the finalizer and stop, nothing else runs
	if !w.DeletionTimestamp.IsZero() {
		// no finalizer = cleanup is already done, nothing to do
		if !controllerutil.ContainsFinalizer(&w, fin) {
			return ctrl.Result{}, nil
		}

		// clean up what lives outside the cluster first, ex: AWS
		if err := r.Shield.Unprotect(ctx, name); err != nil {
			return ctrl.Result{}, err // stays Terminating until this works
		}

		// remove the finalizer last, only after the cleanup worked,
		// then Kubernetes deletes the object.
		// It is only ever removed when the object is deleted.
		controllerutil.RemoveFinalizer(&w, fin)

		// r.Update saves metadata (the finalizer is gone), status is ignored
		return ctrl.Result{}, r.Update(ctx, &w)
	}

	// ### If not being deleted, continue with the rest of the logic... ###

	// #### Update metadata, do the work ####

	// # 3. G3: add the finalizer before you create anything outside the cluster,
	//    on delete the object stays (Terminating) until you remove it, so you get a chance to clean up.
	if controllerutil.AddFinalizer(&w, fin) {
		// r.Update saves metadata (the new finalizer), status is ignored
		if err := r.Update(ctx, &w); err != nil {
			return ctrl.Result{}, err // 409 conflict, stale object, next run reads it fresh
		}
	}

	// # 4. G3: do whatever you need to do outside the cluster, ex: calling third-party APIs...
	if err := r.Shield.Protect(ctx, name); err != nil {
		return ctrl.Result{}, err
	}

	// # 5. G3: create or update what lives inside the cluster, manipulate kubernetes objects, ex: a ConfigMap,
	//    with an owner reference, so Kubernetes deletes it together with your custom resource...
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: w.Name, Namespace: w.Namespace}}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Data = map[string]string{"config": strings.Join(w.Spec.Fragments, "\n")}
		return controllerutil.SetControllerReference(&w, cm, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	// #### Update status ####

	// # 6. G3: set status in memory, nothing is saved yet
	//    after any r.Update, it overwrites w.Status
	meta.SetStatusCondition(&w.Status.Conditions, metav1.Condition{
		Type:               "Available",
		Status:             metav1.ConditionTrue,
		Reason:             "ConfigMerged",
		Message:            fmt.Sprintf("%d fragments merged", len(w.Spec.Fragments)),
		ObservedGeneration: w.Generation,
	})

	w.Status.ObservedGeneration = w.Generation

	// # 7. G3: save the status last, spec and metadata are ignored
	if err := r.Status().Update(ctx, &w); err != nil {
		return ctrl.Result{}, err
	}

	// #### Finished ####

	// # 8. reconcile again later if nothing watches what you created, ex: AWS
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
