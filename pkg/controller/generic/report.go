package generic

import (
	"fmt"
	"strings"

	"github.com/integr8ly/heimdall/pkg/cluster"
	"github.com/integr8ly/heimdall/pkg/domain"
	"github.com/integr8ly/heimdall/pkg/registry"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Reports contains the generic logic to create reports for the images of objects
// managed by its HeimdallObjectInterface implementation
type Reports struct {
	HeimdallObjectInterface

	clusterImageService  *cluster.ImageService
	registryImageService *registry.ImageService

	resourceName string
}

// MakeGenericReports creates a Reports for a resource with resourceName that's
// access is managed by impl
func MakeGenericReports(
	impl HeimdallObjectInterface,
	clusterImageService *cluster.ImageService,
	registryImageService *registry.ImageService,
	resourceName string,
) *Reports {
	return &Reports{
		HeimdallObjectInterface: impl,
		clusterImageService:     clusterImageService,
		registryImageService:    registryImageService,
		resourceName:            resourceName,
	}
}

// GetImages gets a list of images used by obj
func (r *Reports) GetImages(obj v1.Object) ([]*domain.ClusterImage, error) {
	images, err := r.clusterImageService.FindImagesFromLabels(
		obj.GetNamespace(),
		r.GetPodTemplateLabels(obj),
	)
	if err == nil {
		return images, nil
	}

	return nil, errors.Wrapf(
		err,
		"failed to get images for %s in namespace %s",
		r.resourceName,
		obj.GetNamespace(),
	)
}

// Generate generates a report for an object with a given name and namespace,
// delegating the object access logic to r's HeimdallObjectInterface
func (r *Reports) Generate(namespace, name string) ([]domain.ReportResult, error) {
	var reports []domain.ReportResult
	var objects []v1.Object
	checked := map[string]domain.ReportResult{}

	if name == "*" {
		objectsList, err := r.ListObjects(namespace)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list %s objects in namespace %s",
				r.resourceName,
				namespace,
			)
		}
		objects = objectsList
	} else {
		object, err := r.GetObject(namespace, name)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get %s %s in namespace %s",
				r.resourceName,
				name,
				namespace,
			)
		}
		objects = []v1.Object{object}
	}

	for _, obj := range objects {
		images, err := r.GetImages(obj)
		if err != nil {
			fmt.Print("error finding images", err)
		}

		for _, i := range images {
			if !strings.Contains(i.FullPath, "redhat") {
				continue
			}

			if rep, ok := checked[i.SHA256Path]; ok {
				rep.Component = obj.GetName()
				reports = append(reports, rep)
				continue
			}

			result, err := r.registryImageService.Check(i)
			if err != nil {
				fmt.Println(err, "a report failed ")
			} else {
				result.Component = obj.GetName()
				reports = append(reports, result)
			}

			checked[i.FullPath] = result
		}
	}

	return reports, nil
}
