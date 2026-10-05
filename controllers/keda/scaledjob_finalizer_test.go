/*
Copyright 2026 The KEDA Authors

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

package keda

import (
	"context"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	eventingv1alpha1 "github.com/kedacore/keda/v2/apis/eventing/v1alpha1"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/mock/mock_client"
	"github.com/kedacore/keda/v2/pkg/mock/mock_eventemitter"
	"github.com/kedacore/keda/v2/pkg/mock/mock_scaling"
)

var _ = Describe("ScaledJobFinalizer", func() {
	var (
		reconciler   ScaledJobReconciler
		client       *mock_client.MockClient
		scaleHandler *mock_scaling.MockScaleHandler
		eventEmitter *mock_eventemitter.MockEventHandler
		logger       logr.Logger
		ctrl         *gomock.Controller
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		client = mock_client.NewMockClient(ctrl)
		scaleHandler = mock_scaling.NewMockScaleHandler(ctrl)
		eventEmitter = mock_eventemitter.NewMockEventHandler(ctrl)
		logger = logr.Discard()
		reconciler = ScaledJobReconciler{
			Client:       client,
			EventEmitter: eventEmitter,
			scaleHandler: scaleHandler,
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("emits the removal event with the ScaledJob's namespace", func() {
		scaledJob := &kedav1alpha1.ScaledJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "my-sj",
				Namespace:  "my-ns",
				Finalizers: []string{scaledJobFinalizer},
			},
			Spec: kedav1alpha1.ScaledJobSpec{
				Triggers: []kedav1alpha1.ScaleTriggers{{Type: "cron"}},
			},
		}

		scaleHandler.EXPECT().DeleteScalableObject(gomock.Any(), gomock.Any()).Return(nil)
		client.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

		var emittedNamespace string
		eventEmitter.EXPECT().Emit(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(_ runtime.Object, namespace, _ string, _ eventingv1alpha1.CloudEventType, _, _ string) {
				emittedNamespace = namespace
			})

		err := reconciler.finalizeScaledJob(context.Background(), logger, scaledJob,
			types.NamespacedName{Namespace: "my-ns", Name: "my-sj"}.String())

		Expect(err).ToNot(HaveOccurred())
		Expect(emittedNamespace).To(Equal("my-ns"))
	})
})
