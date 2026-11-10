package podzone

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const testNamespace = "default"

// No envtest here: these tests call Reconcile directly on the fake client.
var ctx = context.Background()

func TestPodzone(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Podzone Suite")
}
