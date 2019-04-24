package v1alpha1

import (
	"github.com/kedacore/keda/pkg/apis/keda"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: keda.GroupName, Version: "v1alpha1"}

// Kind takes an unqualified kind and returtns back a Group qualified GroupResource
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	// SchemeBuilder is a SchemaBuilder
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme is an AddToSchema
	AddToScheme = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&ScaledObject{},
		&ScaledObjectList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}