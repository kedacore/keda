package v1alpha1

import (
	"github.com/Azure/Kore/knative/pkg/apis"
	korev1alpha1 "github.com/Azure/Kore/pkg/apis/kore/v1alpha1"
)

func init() {
	apis.AddToSchemes = append(apis.AddToSchemes, korev1alpha1.SchemeBuilder.AddToScheme)
}
