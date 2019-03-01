package controller

import (
	"github.com/Azure/Kore/knative/pkg/controller/podautoscaler"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, podautoscaler.Add)
}
