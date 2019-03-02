package v1alpha1

import (
	"github.com/Azure/Kore/knative/pkg/apis"
	scalingv1alpha1 "github.com/knative/serving/pkg/apis/autoscaling/v1alpha1"
)

func init() {
	apis.AddToSchemes = append(apis.AddToSchemes, scalingv1alpha1.SchemeBuilder.AddToScheme)
}
