package cluster

import (
	"context"
	"fmt"
	"regexp"

	"github.com/integr8ly/heimdall/pkg/domain"
	v1 "github.com/openshift/api/apps/v1"
	"github.com/pkg/errors"
	v12 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ObjectsLabeler struct {
	client client.Client
}

func NewObjectLabeler(c client.Client) *ObjectsLabeler {
	return &ObjectsLabeler{
		client: c,
	}
}

func (ol *ObjectsLabeler) LabelObjects(ctx context.Context, labels map[string]string, excludePattern string, ns string) error {
	dcList := &v1.DeploymentConfigList{}
	depList := &v12.DeploymentList{}
	statSetList := &v12.StatefulSetList{}
	var listOpts = &client.ListOptions{Namespace: ns}
	var matchRegex *regexp.Regexp

	if err := ol.client.List(ctx, dcList, listOpts); err != nil {
		return errors.Wrap(err, "failed to list deployment configs in namespace "+ns)
	}
	if err := ol.client.List(ctx, depList, listOpts); err != nil {
		return errors.Wrap(err, "failed to list deployments  in namespace "+ns)
	}
	if err := ol.client.List(ctx, statSetList, listOpts); err != nil {
		return errors.Wrap(err, "failed to list stateful sets in namespace "+ns)
	}

	var err error
	matchRegex, err = regexp.Compile(excludePattern)
	if err != nil {
		return errors.Wrap(err, "failed to compile pattern to exclude "+excludePattern)
	}

	for _, dc := range dcList.Items {
		// dont care about over writing as these will be our namespaced labels
		for k, v := range labels {
			if excludePattern != "" {
				if matchRegex.MatchString(dc.Name) {
					// ensure the label is not on the dc
					delete(dc.Labels, k)

					continue
				}
			}
			dc.Labels[k] = v
		}
		if err := ol.client.Update(context.TODO(), &dc); err != nil {
			return err
		}
	}
	for _, dep := range depList.Items {
		if dep.Labels == nil {
			dep.Labels = map[string]string{}
		}

		for k, v := range labels {
			if excludePattern != "" {
				if matchRegex.MatchString(dep.Name) {
					fmt.Println("skipping ", dep.Name, " as matched by excludePattern "+excludePattern)
					delete(dep.Labels, k)
					continue
				}
			}
			dep.Labels[k] = v
		}
		if err := ol.client.Update(context.TODO(), &dep); err != nil {
			return err
		}
	}
	for _, statSet := range statSetList.Items {
		if statSet.Labels == nil {
			statSet.Labels = map[string]string{}
		}

		for k, v := range labels {
			if excludePattern != "" && matchRegex.MatchString(statSet.Name) {
				fmt.Printf("skipping %s as matched by excludePattern %s/n",
					statSet.Name, excludePattern)
				delete(statSet.Labels, k)
				continue
			}
			statSet.Labels[k] = v
		}
		if err := ol.client.Update(context.TODO(), &statSet); err != nil {
			return err
		}
	}
	return nil
}

func (ol *ObjectsLabeler) RemoveLabelsAnnotations(ctx context.Context, labels map[string]string, ns string) error {
	dcList := &v1.DeploymentConfigList{}
	depList := &v12.DeploymentList{}
	statSetList := &v12.StatefulSetList{}
	var listOpts = &client.ListOptions{Namespace: ns}

	if err := ol.client.List(ctx, dcList, listOpts); err != nil {
		return errors.Wrap(err, "failed to list deployment configs in namespace "+ns)
	}
	if err := ol.client.List(ctx, depList, listOpts); err != nil {
		return errors.Wrap(err, "failed to list deployments  in namespace "+ns)
	}
	if err := ol.client.List(ctx, statSetList, listOpts); err != nil {
		return errors.Wrap(err, "failed to list stateful sets in namespace "+ns)
	}

	for _, dc := range dcList.Items {

		if dc.Labels != nil {
			for k, _ := range labels {
				delete(dc.Labels, k)
			}
		}
		if dc.Annotations != nil {
			delete(dc.Annotations, domain.HeimdallLastChecked)
			delete(dc.Annotations, domain.HeimdallImagesChecked)
		}

		if err := ol.client.Update(context.TODO(), &dc); err != nil {
			return err
		}
	}
	for _, dep := range depList.Items {
		if dep.Labels != nil {
			for k, _ := range labels {
				delete(dep.Labels, k)
			}
		}
		if dep.Annotations != nil {
			delete(dep.Annotations, domain.HeimdallLastChecked)
			delete(dep.Annotations, domain.HeimdallImagesChecked)
		}
		if err := ol.client.Update(context.TODO(), &dep); err != nil {
			return err
		}
	}
	for _, statSet := range statSetList.Items {
		if statSet.Labels != nil {
			for k, _ := range labels {
				delete(statSet.Labels, k)
			}
		}
		if statSet.Annotations != nil {
			delete(statSet.Annotations, domain.HeimdallLastChecked)
			delete(statSet.Annotations, domain.HeimdallImagesChecked)
		}
		if err := ol.client.Update(context.TODO(), &statSet); err != nil {
			return err
		}
	}
	return nil
}
