// 🌩 Go++ 🌩 #35 · Kubernetes Operators: Cleanup
package widget

import "context"

// Shield is AWS Shield: state outside the cluster that no owner reference
// cleans up. Both calls must be idempotent, a crash means they run again.
type Shield struct{}

func (Shield) Protect(ctx context.Context, name string) error {
	return nil // TODO: call AWS Shield, "already protected" counts as success
}

func (Shield) Unprotect(ctx context.Context, name string) error {
	return nil // TODO: call AWS Shield, "not found" counts as success
}
