package controller

import (
	"github.com/integr8ly/heimdall/pkg/controller/deploymentconfigs"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, deploymentconfigs.Add)
}
