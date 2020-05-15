package statefulset

import (
	"github.com/integr8ly/heimdall/pkg/cluster"
	"github.com/integr8ly/heimdall/pkg/controller/generic"
	"github.com/integr8ly/heimdall/pkg/registry"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

// NewReport creates a generic.Reports that generates reports for stateful sets
func NewReport(clusterImageService *cluster.ImageService, registryImageService *registry.ImageService, client v1.AppsV1Interface) *generic.Reports {
	return generic.MakeGenericReports(
		&objectInterface{client},
		clusterImageService,
		registryImageService,
		"stateful set",
	)
}
