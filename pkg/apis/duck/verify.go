package duck

import (
	"encoding/json"

	runtime "k8s.io/apimachinery/pkg/runtime"
)

// Implementable is implemented by the Fooable duck type that consumers
// are expected to embed as a `.status.fooable` field.
type Implementable interface {
	// GetFullType returns an instance of a full resource wrapping
	// an instance of this Implementable that can populate its fields
	// to verify json roundtripping.
	GetFullType() Populatable
}

// Populatable is implemented by a skeleton resource wrapping an Implementable
// duck type.  It will generally have TypeMeta, ObjectMeta, and a Status field
// wrapping a Fooable field.
type Populatable interface {
	Listable

	// Populate fills in all possible fields, so that we can verify that
	// they roundtrip properly through JSON.
	Populate()
}

// Listable indicates that a particular type can be returned via the returned
// list type by the API server.
type Listable interface {
	runtime.Object

	GetListType() runtime.Object
}

// FromUnstructured takes unstructured object from (say from client-go/dynamic) and
// converts it into our duck types.
func FromUnstructured(obj json.Marshaler, target interface{}) error {
	// Use the unstructured marshaller to ensure it's proper JSON
	raw, err := obj.MarshalJSON()
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, &target)
}
