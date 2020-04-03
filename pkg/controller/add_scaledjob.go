package controller

import (
	"github.com/kedacore/keda/pkg/controller/scaledjob"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, scaledjob.Add)
}
