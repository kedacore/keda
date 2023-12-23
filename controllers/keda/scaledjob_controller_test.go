package keda

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

var _ = Describe("ScaledJobController", func() {

	var (
		testLogger = zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter))
	)

	Describe("functional tests", func() {
		It("scaledjob paused condition status changes to true on annotation", func() {
			jobName := "toggled-to-paused-annotation-name"
			sjName := "sj-" + jobName

			sj := &kedav1alpha1.ScaledJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sjName,
					Namespace: "default",
				},
				Spec: kedav1alpha1.ScaledJobSpec{
					JobTargetRef: generateJobSpec(jobName),
					Triggers: []kedav1alpha1.ScaleTriggers{
						{
							Type: "cron",
							Metadata: map[string]string{
								"timezone":        "UTC",
								"start":           "0 * * * *",
								"end":             "1 * * * *",
								"desiredReplicas": "1",
							},
						},
					},
				},
			}
			pollingInterval := int32(5)
			sj.Spec.PollingInterval = &pollingInterval
			err := k8sClient.Create(context.Background(), sj)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() metav1.ConditionStatus {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: sjName, Namespace: "default"}, sj)
				if err != nil {
					return metav1.ConditionTrue
				}
				return sj.Status.Conditions.GetPausedCondition().Status
			}, 5*time.Second).Should(Or(Equal(metav1.ConditionFalse), Equal(metav1.ConditionUnknown)))

			// set annotation
			Eventually(func() error {
				err = k8sClient.Get(context.Background(), types.NamespacedName{Name: sjName, Namespace: "default"}, sj)
				Expect(err).ToNot(HaveOccurred())
				annotations := make(map[string]string)
				annotations[kedav1alpha1.PausedAnnotation] = "true"
				sj.SetAnnotations(annotations)
				pollingInterval := int32(6)
				sj.Spec.PollingInterval = &pollingInterval
				return k8sClient.Update(context.Background(), sj)
			}).WithTimeout(1 * time.Minute).WithPolling(10 * time.Second).ShouldNot(HaveOccurred())
			testLogger.Info("annotation is set")

			// validate annotation is set correctly
			Eventually(func() bool {
				err = k8sClient.Get(context.Background(), types.NamespacedName{Name: sjName, Namespace: "default"}, sj)
				Expect(err).ToNot(HaveOccurred())
				_, hasAnnotation := sj.GetAnnotations()[kedav1alpha1.PausedAnnotation]
				return hasAnnotation
			}).WithTimeout(1 * time.Minute).WithPolling(2 * time.Second).Should(BeTrue())

			Eventually(func() metav1.ConditionStatus {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: sjName, Namespace: "default"}, sj)
				if err != nil {
					return metav1.ConditionUnknown
				}
				return sj.Status.Conditions.GetPausedCondition().Status
			}).WithTimeout(2 * time.Minute).WithPolling(10 * time.Second).Should(Equal(metav1.ConditionTrue))
		})
		It("scaledjob paused status stays false when annotation is set to false", func() {
			jobName := "turn-off-paused-annotation-name"
			sjName := "sj-" + jobName
			// create object already paused
			sj := &kedav1alpha1.ScaledJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sjName,
					Namespace: "default",
				},
				Spec: kedav1alpha1.ScaledJobSpec{
					JobTargetRef: generateJobSpec(jobName),
					Triggers: []kedav1alpha1.ScaleTriggers{
						{
							Type: "cron",
							Metadata: map[string]string{
								"timezone":        "UTC",
								"start":           "0 * * * *",
								"end":             "1 * * * *",
								"desiredReplicas": "1",
							},
						},
					},
				},
			}
			pollingInterval := int32(5)
			sj.Spec.PollingInterval = &pollingInterval
			err := k8sClient.Create(context.Background(), sj)
			Expect(err).ToNot(HaveOccurred())
			falseAnnotationValue := "false"
			// set annotation
			Eventually(func() error {
				err = k8sClient.Get(context.Background(), types.NamespacedName{Name: sjName, Namespace: "default"}, sj)
				Expect(err).ToNot(HaveOccurred())
				annotations := make(map[string]string)
				annotations[kedav1alpha1.PausedAnnotation] = falseAnnotationValue
				sj.SetAnnotations(annotations)
				pollingInterval := int32(6)
				sj.Spec.PollingInterval = &pollingInterval
				return k8sClient.Update(context.Background(), sj)
			}).WithTimeout(1 * time.Minute).WithPolling(10 * time.Second).ShouldNot(HaveOccurred())
			testLogger.Info("annotation is set")

			// validate annotation is set correctly
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: sjName, Namespace: "default"}, sj)
				Expect(err).ToNot(HaveOccurred())
				value, hasPausedAnnotation := sj.GetAnnotations()[kedav1alpha1.PausedAnnotation]
				if !hasPausedAnnotation {
					return false
				}
				return value == falseAnnotationValue
			}).WithTimeout(1 * time.Minute).WithPolling(2 * time.Second).Should(BeTrue())

			// TODO(nappelson) - update assertion to be ConditionFalse
			// https://github.com/kedacore/keda/issues/5251 prevents Condition from updating appropriately
			Eventually(func() metav1.ConditionStatus {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: sjName, Namespace: "default"}, sj)
				if err != nil {
					return metav1.ConditionUnknown
				}
				return sj.Status.Conditions.GetPausedCondition().Status
			}).WithTimeout(1 * time.Minute).WithPolling(10 * time.Second).Should(Equal(metav1.ConditionUnknown))
		})

	})
})

func generateJobSpec(name string) *batchv1.JobSpec {
	return &batchv1.JobSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": name,
			},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app": name,
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  name,
						Image: name,
					},
				},
			},
		},
	}
}
