package cluster

import (
	"context"
	"fmt"
	"github.com/integr8ly/heimdall/pkg/domain"
	v1 "github.com/openshift/api/apps/v1"
	"github.com/pkg/errors"
	v12 "k8s.io/api/apps/v1"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ObjectsLabeler struct {
	client client.Client
}

func NewObjectLabeler(c client.Client)*ObjectsLabeler  {
	return &ObjectsLabeler{
		client:c,
	}
}

func (ol *ObjectsLabeler)LabelAllDeploymentsAndDeploymentConfigs(ctx context.Context,labels map[string]string , excludePattern string, ns string) error {
	dcList := &v1.DeploymentConfigList{}
	depList := &v12.DeploymentList{}
	var listOpts = &client.ListOptions{Namespace:ns}
	var matchRegex *regexp.Regexp

	if err := ol.client.List(ctx,listOpts,dcList); err != nil{
		return errors.Wrap(err, "failed to list deployment configs in namespace " + ns)
	}
	if err := ol.client.List(ctx, listOpts, depList); err != nil{
		return errors.Wrap(err, "failed to list deployments  in namespace " + ns)
	}

		var err error
		matchRegex, err = regexp.Compile(excludePattern)
		if err != nil{
			return errors.Wrap(err, "failed to compile pattern to exclude " + excludePattern)
		}

		for _, dc := range dcList.Items{
			if excludePattern != "" {
				if matchRegex.MatchString(dc.Name) {
					fmt.Println("skipping ", dc.Name, " as matched by excludePattern"+excludePattern)
					continue
				}
			}
			// dont care about over writing as these will be our namespaced labels
			for k, v :=  range labels{
				dc.Labels[k] =v
			}
			if err := ol.client.Update(context.TODO(), &dc); err != nil{
				return err
			}
		}
		for _, dep := range depList.Items{
			if dep.Labels == nil{
				dep.Labels = map[string]string{}
			}
			if matchRegex.MatchString(dep.Name){
				fmt.Println("skipping ", dep.Name, " as matched by excludePattern " + excludePattern)
				continue
			}
			for k, v :=  range labels{
				dep.Labels[k] =v
			}
			if err := ol.client.Update(context.TODO(), &dep); err != nil{
				return err
			}
		}
	return nil
}

func (ol *ObjectsLabeler)RemoveLabelsAnnotations(ctx context.Context,labels map[string]string , excludePattern string, ns string) error {
	dcList := &v1.DeploymentConfigList{}
	depList := &v12.DeploymentList{}
	var listOpts = &client.ListOptions{Namespace:ns}
	var matchRegex *regexp.Regexp



	if err := ol.client.List(ctx,listOpts,dcList); err != nil{
		return errors.Wrap(err, "failed to list deployment configs in namespace " + ns)
	}
	if err := ol.client.List(ctx, listOpts, depList); err != nil{
		return errors.Wrap(err, "failed to list deployments  in namespace " + ns)
	}

	var err error
	matchRegex, err = regexp.Compile(excludePattern)
	if err != nil{
		return errors.Wrap(err, "failed to compile pattern to exclude " + excludePattern)
	}


	for _, dc := range dcList.Items{
		if matchRegex.MatchString(dc.Name){
			fmt.Println("skipping ", dc.Name, " as matched by excludePattern")
			continue
		}

		if dc.Labels != nil {
			for k, _ := range labels {
				delete(dc.Labels, k)
			}
		}
		if dc.Annotations != nil{
			delete(dc.Annotations, domain.HeimdallLastChecked)
			delete(dc.Annotations, domain.HeimdallImagesChecked)
		}

		if err := ol.client.Update(context.TODO(), &dc); err != nil{
			return err
		}
	}
	for _, dep := range depList.Items{

		if matchRegex.MatchString(dep.Name){
			fmt.Println("skipping ", dep.Name, " as matched by excludePattern")
			continue
		}
		if dep.Labels != nil {
			for k, _ := range labels {
				delete(dep.Labels, k)
			}
		}
		if dep.Annotations != nil{
			delete(dep.Annotations, domain.HeimdallLastChecked)
			delete(dep.Annotations, domain.HeimdallImagesChecked)
		}
		if err := ol.client.Update(context.TODO(), &dep); err != nil{
			return err
		}
	}
	return nil
}