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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// The fast way to test Reconcile's logic: the fake client, called directly,
// with no API server, cache or workqueue behind it.
var _ = Describe("Pod Controller", func() {
	It("copies the node's zone onto a scheduled Pod", func() {
		node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{
			Name:   "node-a",
			Labels: map[string]string{corev1.LabelTopologyZone: "eu-west-1a"},
		}}
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "api-0", Namespace: testNamespace},
			Spec:       corev1.PodSpec{NodeName: "node-a"},
		}
		c := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(node, pod).Build()
		r := &PodReconciler{Client: c, Scheme: scheme.Scheme}
		key := client.ObjectKeyFromObject(pod)

		_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key})
		Expect(err).NotTo(HaveOccurred())

		Expect(c.Get(ctx, key, pod)).To(Succeed())
		Expect(pod.Labels).To(HaveKeyWithValue(corev1.LabelTopologyZone, "eu-west-1a"))
	})

	It("leaves an unscheduled Pod alone until it is bound", func() {
		pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "api-1", Namespace: testNamespace}}
		c := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(pod).Build()
		r := &PodReconciler{Client: c, Scheme: scheme.Scheme}

		_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(pod)})
		Expect(err).NotTo(HaveOccurred())
	})
})
