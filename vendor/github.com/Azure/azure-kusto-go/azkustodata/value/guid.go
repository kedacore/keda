package value

import (
	"github.com/Azure/azure-kusto-go/azkustodata/types"
	"reflect"

	"github.com/google/uuid"
)

// GUID represents a Kusto GUID type.  GUID implements Kusto.
type GUID struct {
	pointerValue[uuid.UUID]
}

// NewGUID creates a new GUID.
func NewGUID(v uuid.UUID) *GUID { return &GUID{newPointerValue[uuid.UUID](&v)} }

// NewNullGUID creates a new null GUID.
func NewNullGUID() *GUID { return &GUID{newPointerValue[uuid.UUID](nil)} }

// Unmarshal unmarshals i into GUID. i must be a string representing a GUID or nil.
func (g *GUID) Unmarshal(i interface{}) error {
	if i == nil {
		g.value = nil
		return nil
	}
	str, ok := i.(string)
	if !ok {
		return convertError(g, i)
	}
	u, err := uuid.Parse(str)
	if err != nil {
		return parseError(g, i, err)
	}

	g.value = &u
	return nil
}

// Convert GUID into reflect value.
func (g *GUID) Convert(v reflect.Value) error {
	return Convert[uuid.UUID](*g, &g.pointerValue, v)
}

// GetType returns the type of the value.
func (g *GUID) GetType() types.Column {
	return types.GUID
}
