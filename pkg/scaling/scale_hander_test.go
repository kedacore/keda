package scaling

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestTargetAverageValue(t *testing.T) {
	// count = 0
	specs := []v2beta2.MetricSpec{}
	targetAverageValue := getTargetAverageValue(specs)
	assert.Equal(t, int64(0), targetAverageValue)
	// 1 1
	specs = []v2beta2.MetricSpec{
		createMetricSpec(1),
		createMetricSpec(1),
	}
	targetAverageValue = getTargetAverageValue(specs)
	assert.Equal(t, int64(1), targetAverageValue)
	// 5 5 3
	specs = []v2beta2.MetricSpec{
		createMetricSpec(5),
		createMetricSpec(5),
		createMetricSpec(3),
	}
	targetAverageValue = getTargetAverageValue(specs)
	assert.Equal(t, int64(4), targetAverageValue)

	// 5 5 4
	specs = []v2beta2.MetricSpec{
		createMetricSpec(5),
		createMetricSpec(5),
		createMetricSpec(3),
	}
	targetAverageValue = getTargetAverageValue(specs)
	assert.Equal(t, int64(4), targetAverageValue)
}

func createMetricSpec(averageValue int) v2beta2.MetricSpec {
	qty := resource.NewQuantity(int64(averageValue), resource.DecimalSI)
	return v2beta2.MetricSpec{
		External: &v2beta2.ExternalMetricSource{
			Target: v2beta2.MetricTarget{
				AverageValue: qty,
			},
		},
	}
}

func TestDuckFromRuntimeObjectDeployment(t *testing.T) {
	withPods := &duckv1.WithPod{}
	// Test with a Deployment.
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "main",
						},
					},
				},
			},
		},
	}
	err := duckFromRuntimeObject(deployment, withPods)
	assert.Nil(t, err)
	assert.Equal(t, withPods.ObjectMeta.Name, "foo")
	assert.Equal(t, withPods.ObjectMeta.Namespace, "bar")
	assert.Equal(t, withPods.Spec.Template.Spec.Containers[0].Name, "main")
}

func TestDuckFromRuntimeObjectStatefulSet(t *testing.T) {
	withPods := &duckv1.WithPod{}
	// Test with a StatefulSet.
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "main",
						},
					},
				},
			},
		},
	}
	err := duckFromRuntimeObject(statefulSet, withPods)
	assert.Nil(t, err)
	assert.Equal(t, withPods.ObjectMeta.Name, "foo")
	assert.Equal(t, withPods.ObjectMeta.Namespace, "bar")
	assert.Equal(t, withPods.Spec.Template.Spec.Containers[0].Name, "main")
}
