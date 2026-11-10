// 🌩 Go++ 🌩 #35 · Kubernetes Operators: Cleanup
package widget

import (
	"context"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// Shield is the AWS Shield API as the Widget controller needs it: state
// outside the cluster that no owner reference will ever clean up.
// Defined here, where it is used, not next to an implementation.
//
// Both calls must be idempotent. The controller can crash between Unprotect
// and removing its finalizer, and will call Unprotect again.
type Shield interface {
	Protect(ctx context.Context, name string) error
	Unprotect(ctx context.Context, name string) error
}

// LogShield stands in for AWS in this example: it logs instead of calling out.
type LogShield struct{}

func (LogShield) Protect(ctx context.Context, name string) error {
	logf.FromContext(ctx).V(1).Info("shield: protect", "name", name)
	return nil
}

func (LogShield) Unprotect(ctx context.Context, name string) error {
	logf.FromContext(ctx).Info("shield: unprotect", "name", name)
	return nil
}
