package cluster

import (
	"context"
	"fmt"
	"github.com/integr8ly/heimdall/pkg/domain"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	LabelContainerFormat                 = "heimdall.%s.%s"
	labelAggregateFormat                 = "heimdall.%s"
	LabelAggregateResolvableCritCVE      = "heimdall.resolvableCriticalCVEs"
	LabelAggregateResolvableImportantCVE = "heimdall.resolvableImportantCVEs"
	LabelAggregateResolvableModerateCVE  = "heimdall.resolvableModerateCVEs"
)

type Pods struct {
	client client.Client
}

func NewPods(c client.Client) *Pods {
	return &Pods{client: c}
}

func (p *Pods) LabelPods(rep *domain.ReportResult) error {

	var labelErrors = []error{}
	for _, pd := range rep.ClusterImage.Pods {
		log.Info("labeling pod with image info ", "pod ", pd.Name, "namespace", pd.Namespace)
		pod := &v1.Pod{}
		if err := p.client.Get(context.TODO(), client.ObjectKey{Name: pd.Name, Namespace: pd.Namespace}, pod); err != nil {
			log.Error(err, "failed to get the pod "+pd.Name+" in namespace "+pd.Namespace+" will try again later")
			labelErrors = append(labelErrors, err)
			continue
		}
		if pod.Labels == nil {
			pod.Labels = map[string]string{}
		}
		if pod.Annotations == nil {
			pod.Annotations = map[string]string{}
		}
		if pod.Status.Phase != v1.PodRunning {
			log.Info("not labeling pod as it is not running")
			return errors.New("could not label pod " + pod.Name + " in namespace " + pod.Namespace + " as it is not in a running phase")
		}

		pod.Labels[LabelAggregateResolvableImportantCVE] = fmt.Sprintf("%v", len(rep.GetResolvableImportantCVEs()) > 0)
		pod.Labels[LabelAggregateResolvableCritCVE] = fmt.Sprintf("%v", len(rep.GetResolvableCriticalCVEs()) > 0)
		pod.Labels[LabelAggregateResolvableModerateCVE] = fmt.Sprintf("%v", len(rep.GetResolvableModerateCVEs()) > 0)

		for _, c := range pd.Containers {
			pod.Labels[fmt.Sprintf(LabelContainerFormat, c, "resolvableImportantCVEs")] = fmt.Sprintf("%v", len(rep.GetResolvableImportantCVEs()))
			pod.Labels[fmt.Sprintf(LabelContainerFormat, c, "resolvableCriticalCVEs")] = fmt.Sprintf("%v", len(rep.GetResolvableCriticalCVEs()))
			pod.Labels[fmt.Sprintf(LabelContainerFormat, c, "resolvableModerateCVEs")] = fmt.Sprintf("%v", len(rep.GetResolvableModerateCVEs()))
			pod.Labels[fmt.Sprintf(labelAggregateFormat, "updatedImageAvailable")] = fmt.Sprintf("%v", rep.UpToDateWithFloatingTag == false)
			pod.Labels[fmt.Sprintf(LabelContainerFormat, c, "latestPatchImage")] = fmt.Sprintf("%v", rep.LatestAvailablePatchVersion)
			pod.Labels[fmt.Sprintf(LabelContainerFormat, c, "currentImage")] = fmt.Sprintf("%v", rep.CurrentVersion)
			if rep.ClusterImage.FromImageStream {
				pod.Annotations[fmt.Sprintf(LabelContainerFormat, c, "imagestreamTag")] = fmt.Sprintf("%v", rep.ClusterImage.ImageStreamTag.Name)
				pod.Annotations[fmt.Sprintf(LabelContainerFormat, c, "imagestreamTagNamespace")] = fmt.Sprintf("%v", rep.ClusterImage.ImageStreamTag.Namespace)
			}
			if err := p.client.Update(context.TODO(), pod); err != nil {
				labelErrors = append(labelErrors, errors.Wrap(err, "failed to update pod with labels"))
				continue
			}
		}
	}
	if len(labelErrors) > 0 {
		errMsg := ""
		for _, e := range labelErrors {
			errMsg += e.Error() + " "
		}
		return errors.New(errMsg)
	}
	return nil
}
