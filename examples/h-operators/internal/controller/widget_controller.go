/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

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
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	goplusplusv1 "github.com/semirm-dev/goplusplus/examples/h-operators/api/v1"
)

// Domain-qualified, or the API server warns on every write.
const shieldFinalizer = "goplusplus.semirmahovkic.xyz/shield-protection"

// WidgetReconciler merges a Widget's fragments into a ConfigMap it owns, and
// keeps the Widget protected by Shield for as long as the Widget exists.
type WidgetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Shield Shield
}

// +kubebuilder:rbac:groups=goplusplus.semirmahovkic.xyz,resources=widgets,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=goplusplus.semirmahovkic.xyz,resources=widgets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=goplusplus.semirmahovkic.xyz,resources=widgets/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch

// Reconcile gets a key, not an event: it reads the Widget fresh and converges,
// so running it once or fifty times lands in the same place.
func (r *WidgetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var w goplusplusv1.Widget
	if err := r.Get(ctx, req.NamespacedName, &w); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err) // gone: nothing left to converge
	}

	// Being deleted: clean up outside the cluster, then let go.
	if !w.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&w, shieldFinalizer) {
			return ctrl.Result{}, nil
		}
		if err := r.Shield.Unprotect(ctx, req.String()); err != nil {
			return ctrl.Result{}, err // the Widget stays Terminating until this succeeds
		}
		controllerutil.RemoveFinalizer(&w, shieldFinalizer)
		return ctrl.Result{}, r.Update(ctx, &w)
	}

	// Finalizer first, then the external resource. The other order leaks a
	// protection if the Widget is deleted between the two.
	if controllerutil.AddFinalizer(&w, shieldFinalizer) {
		if err := r.Update(ctx, &w); err != nil {
			return ctrl.Result{}, err // a conflict too: requeued, read fresh
		}
	}
	if err := r.Shield.Protect(ctx, req.String()); err != nil {
		return ctrl.Result{}, err
	}

	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: w.Name, Namespace: w.Namespace}}
	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Data = map[string]string{"config": strings.Join(w.Spec.Fragments, "\n")}
		// Inside the mutate func, so a reference someone stripped is put back too.
		return controllerutil.SetControllerReference(&w, cm, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	log.V(1).Info("configmap", "operation", op)

	// SetStatusCondition only moves LastTransitionTime when Status flips.
	meta.SetStatusCondition(&w.Status.Conditions, metav1.Condition{
		Type:               "Available",
		Status:             metav1.ConditionTrue,
		Reason:             "ConfigMerged",
		Message:            fmt.Sprintf("%d fragments merged", len(w.Spec.Fragments)),
		ObservedGeneration: w.Generation,
	})
	w.Status.ObservedGeneration = w.Generation
	if err := r.Status().Update(ctx, &w); err != nil {
		return ctrl.Result{}, err
	}

	// Nothing watches AWS for us, so a protection removed there is only
	// noticed by looking again.
	return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WidgetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&goplusplusv1.Widget{}).
		Owns(&corev1.ConfigMap{}). // someone edits the ConfigMap -> reconcile its Widget
		Named("widget").
		Complete(r)
}
