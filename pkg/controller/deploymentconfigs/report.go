package deploymentconfigs

import (
	"fmt"
	"github.com/integr8ly/heimdall/pkg/cluster"
	"github.com/integr8ly/heimdall/pkg/domain"
	"github.com/integr8ly/heimdall/pkg/registry"
	v1 "github.com/openshift/api/apps/v1"
	"github.com/pkg/errors"
	v12 "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

func getImageChangeParams(dc *v1.DeploymentConfig)[]*v1.DeploymentTriggerImageChangeParams  {
	var ret []*v1.DeploymentTriggerImageChangeParams
	for _, tr := range dc.Spec.Triggers {
		if tr.Type == v1.DeploymentTriggerOnImageChange && tr.ImageChangeParams != nil {
			ret = append(ret, tr.ImageChangeParams)
		}
	}
	return ret
}


type Reports struct {
	clusterImageService *cluster.ImageService
	registryImageService *registry.ImageService
	dcClient *v12.AppsV1Client
}

func NewReport(clusterImageService *cluster.ImageService,
registryImageService *registry.ImageService, dcClient *v12.AppsV1Client)*Reports  {
	return &Reports{
		clusterImageService:  clusterImageService,
		registryImageService: registryImageService,
		dcClient: dcClient,
	}
}

func (r *Reports)Generate(ns, deploymentConfig string)([]domain.ReportResult,  error)  {
	var dcs []v1.DeploymentConfig

	if deploymentConfig == "*"{
		dcList, err := r.dcClient.DeploymentConfigs(ns).List(v13.ListOptions{})
		if err != nil{
			return nil, errors.Wrap(err, "failed to list deploymentconfigs in namespace " + ns)
		}
		dcs = dcList.Items
	}else{
		dc, err := r.dcClient.DeploymentConfigs(ns).Get(deploymentConfig, v13.GetOptions{})
		if err != nil{
			return nil, errors.Wrap(err, "failed to get deploymentconfigs in namespace " + ns + " with name " + deploymentConfig)
		}
		dcs = append(dcs, *dc)
	}

	var reports []domain.ReportResult
	var checked= map[string]domain.ReportResult{}
	for _, dc := range dcs {
		log.Info("checking deployment with resource version " + dc.ResourceVersion)
		var images []*domain.ClusterImage
		log.Info("got deployment config ", "name", dc.Name)
		icp := getImageChangeParams(&dc)
		if len(icp) > 0 {
			is, err := r.clusterImageService.FindImagesFromImageChangeParams(dc.Namespace, icp, dc.Labels)
			if err != nil {
				return reports, errors.Wrap(err, "failed find images in deploymentconfig via its image triggers ")
			}
			images = append(images, is...)
		} else {
			is, err := r.clusterImageService.FindImagesFromLabels(dc.Namespace, dc.Spec.Template.Labels)
			if err != nil {
				return reports, errors.Wrap(err, "failed find images in deploymentconfig")
			}
			images = append(images, is...)
		}
		log.Info("found images ", "no:", len(images))

		for _, i := range images {
			if !strings.Contains(i.FullPath, "redhat") {
				log.Info("skipping image not in redhat registry " + i.FullPath)
				continue
			}
			if _, ok := checked[i.SHA256Path]; ok {
				log.Info("already checked ", i.SHA256Path, "skipping ")
				rep := checked[i.SHA256Path]
				rep.Component = dc.Name
				reports = append(reports, rep)
				continue
			}
			log.Info("looking at image after parsing ", "image full tag", i.FullPath)
			result, err := r.registryImageService.Check(i)
			if err != nil {
				// log and carry on as we may have valid reports
				fmt.Println("error creating report ", err)
				fmt.Println("REPORTs ", len(reports))
				log.Error(err, "a report failed ")
			}else {
				result.ClusterImage = i
				result.Component = dc.Name
				reports = append(reports, result)
			}
			checked[i.SHA256Path] = result
		}

	}
	return reports, nil
}
