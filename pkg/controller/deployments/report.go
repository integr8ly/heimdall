package deployments

import (
	"fmt"
	"github.com/integr8ly/heimdall/pkg/cluster"
	"github.com/integr8ly/heimdall/pkg/domain"
	"github.com/integr8ly/heimdall/pkg/registry"
	"github.com/pkg/errors"

	v12 "k8s.io/api/apps/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"strings"
)

type Reports struct {
	clusterImageService *cluster.ImageService
	registryImageService *registry.ImageService
	deploymentClient v1.AppsV1Interface
}

func NewReport(clusterImageService *cluster.ImageService,
	registryImageService *registry.ImageService, deploymentClient v1.AppsV1Interface)*Reports  {
	return &Reports{
		clusterImageService:  clusterImageService,
		registryImageService: registryImageService,
		deploymentClient: deploymentClient,
	}
}

func (r *Reports)Generate(ns, name string)([]domain.ReportResult,  error)  {
	var reports []domain.ReportResult
	var deployments []v12.Deployment
	var checked= map[string]domain.ReportResult{}
	if name == "*"{
		dl, err := r.deploymentClient.Deployments(ns).List(v13.ListOptions{})
		if err != nil{
			return nil, errors.Wrap(err, "failed to list deployments in namespace "+ns)
		}
		deployments = dl.Items
	}else{
		d,err := r.deploymentClient.Deployments(ns).Get(name, v13.GetOptions{})
		if err != nil{
			return nil, errors.Wrap(err, fmt.Sprintf("failed to get deployment %s in namespace %s ", name, ns))
		}
		deployments = append(deployments, *d)
	}

	for _, d := range deployments{
		images,err := r.GetImages(&d)
		if err != nil{
			// log
			fmt.Println("error finding images", err)
		}
		for _, i := range images{
			if !strings.Contains(i.FullPath, "redhat") {
				continue
			}
			if _, ok := checked[i.SHA256Path]; ok {
				rep := checked[i.SHA256Path]
				rep.Component = d.Name
				reports = append(reports, rep)
				continue
			}
			result, err := r.registryImageService.Check(i)
			if err != nil {
				// log and carry on as we may have valid reports
				fmt.Println(err, "a report failed ")
			}else {
				reports = append(reports, result)
			}
			result.Component = d.Name
			checked[i.FullPath] = result
		}
	}
	return reports, nil
}

func (r *Reports)GetImages(d *v12.Deployment)([]*domain.ClusterImage, error )  {
	images,err := r.clusterImageService.FindImagesFromLabels(d.Namespace, d.Spec.Template.Labels)
	if err != nil{
		return nil, errors.Wrap(err, "failed to get images for deployment " + d.Name + " in namespace " + d.Namespace)
	}
	return images, nil
}
