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

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// Shield is the AWS Shield API as the Widget controller needs it: state
// outside the cluster that no owner reference will ever clean up.
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
