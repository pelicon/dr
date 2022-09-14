package controller

import (
	"github.com/pelicon/dr/pkg/controller/drcluster"
	"github.com/pelicon/dr/pkg/controller/drnamespace"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, drcluster.Add)

	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, drnamespace.Add)
}
