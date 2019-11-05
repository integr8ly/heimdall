package controller

import (
	"github.com/integr8ly/heimdall/pkg/controller/deploymentconfigs"
	"github.com/integr8ly/heimdall/pkg/controller/deployments"
	"github.com/integr8ly/heimdall/pkg/controller/imagemonitor"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, deploymentconfigs.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, deployments.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, imagemonitor.Add)
}
