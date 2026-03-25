package keda

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
			}, 5*time.Second).Should(Equal(metav1.ConditionFalse))

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
			}).WithTimeout(1 * time.Minute).WithPolling(10 * time.Second).Should(Equal(metav1.ConditionFalse))
		})

		// Fix issue 5520
		It("create scaledjob with empty triggers should be blocked", func() {
			jobName := "empty-triggers-sj-name"
			sjName := "sj-" + jobName
			sj := &kedav1alpha1.ScaledJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sjName,
					Namespace: "default",
				},
				Spec: kedav1alpha1.ScaledJobSpec{
					JobTargetRef: generateJobSpec(jobName),
					Triggers:     []kedav1alpha1.ScaleTriggers{},
				},
			}

			// CRD-level MinItems=1 validation on spec.triggers rejects the
			// request before it reaches the webhook or controller.
			err := k8sClient.Create(context.Background(), sj)
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsInvalid(err)).To(BeTrue())
			statusErr, ok := err.(*apierrors.StatusError)
			Expect(ok).To(BeTrue())
			Expect(statusErr.ErrStatus.Details).ToNot(BeNil())
			Expect(statusErr.ErrStatus.Details.Causes).ToNot(BeEmpty())
			foundTriggersField := false
			for _, cause := range statusErr.ErrStatus.Details.Causes {
				if cause.Field == "spec.triggers" {
					foundTriggersField = true
					break
				}
			}
			Expect(foundTriggersField).To(BeTrue())
		})

		It("ScaledJob minReplicaCount defaults to nil when not set", func() {
			jobName := "use-default-minreplicacount-value"
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

			// Confirm the minReplicaCount is nil
			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: sjName, Namespace: "default"}, sj)
			Expect(err).ToNot(HaveOccurred())
			Expect(sj.Spec.MinReplicaCount).To(BeNil())
		})

		It("ScaledJob maxReplicaCount defaults to nil when not set", func() {
			jobName := "use-default-maxreplicacount-value"
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

			// Confirm the maxReplicaCount is nil
			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: sjName, Namespace: "default"}, sj)
			Expect(err).ToNot(HaveOccurred())
			Expect(sj.Spec.MaxReplicaCount).To(BeNil())
		})

		It("ScaledJob minReplicaCount is set to maxReplicaCount when maxReplicaCount is less than minReplicaCount", func() {
			jobName := "minreplicacount-changes-to-maxreplicacount"
			sjName := "sj-" + jobName
			minReplicaCount := int32(10)
			maxReplicaCount := int32(3)
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
					MinReplicaCount: &minReplicaCount,
					MaxReplicaCount: &maxReplicaCount,
				},
			}
			pollingInterval := int32(5)
			sj.Spec.PollingInterval = &pollingInterval
			err := k8sClient.Create(context.Background(), sj)
			Expect(err).ToNot(HaveOccurred())

			// Confirm that minReplicaCount is set to maxReplicaCount
			Eventually(func() *int32 {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: sjName, Namespace: "default"}, sj)
				Expect(err).ToNot(HaveOccurred())
				return sj.Spec.MinReplicaCount
			}).WithTimeout(1 * time.Minute).WithPolling(5 * time.Second).Should(Equal(&maxReplicaCount))
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
