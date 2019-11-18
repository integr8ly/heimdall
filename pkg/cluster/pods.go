package cluster

import (
	"fmt"
	"github.com/integr8ly/heimdall/pkg/domain"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"context"
)

const labelFormat = "heimdall.%s"

type Pods struct {
	client client.Client
}

func NewPods(c client.Client)*Pods  {
	return &Pods{client:c}
}

func (p *Pods)LabelPods(rep *domain.ReportResult)error  {

	var labelErrors = []error{}
	fmt.Println("report pods ", rep.ClusterImage.Pods)
	for _, pd := range rep.ClusterImage.Pods{
		log.Info("labeling pod with image info ", "pod ", pd.Name, "namespace", pd.Namespace)
		pod := &v1.Pod{}
		if err := p.client.Get(context.TODO(), client.ObjectKey{Name:pd.Name, Namespace:pd.Namespace}, pod); err != nil{
			log.Error(err, "failed to get the pod " + pd.Name + " in namespace " + pd.Namespace + " will try again later")
			labelErrors = append(labelErrors,err)
			continue
		}

		pod.Labels[fmt.Sprintf(labelFormat,"resolvableImportantCVEs")] = fmt.Sprintf("%d",len(rep.GetResolvableImportantCVEs()))
		pod.Labels[fmt.Sprintf(labelFormat,"resolvableCriticalCVEs")] = fmt.Sprintf("%d",len(rep.GetResolvableCriticalCVEs()))
		pod.Labels[fmt.Sprintf(labelFormat,"resolvableModerateCVEs")] = fmt.Sprintf("%d",len(rep.GetResolvableModerateCVEs()))
		pod.Labels[fmt.Sprintf(labelFormat,"updatedImageAvailable")] = fmt.Sprintf("%v",rep.UpToDateWithFloatingTag == false)
		pod.Labels[fmt.Sprintf(labelFormat,"latestPatchImage")] = fmt.Sprintf("%v",rep.LatestAvailablePatchVersion)
		if rep.ClusterImage.FromImageStream {
			pod.Annotations[fmt.Sprintf(labelFormat, "imagestreamTag")] = fmt.Sprintf("%v", rep.ClusterImage.ImageStreamTag.Name)
			pod.Annotations[fmt.Sprintf(labelFormat, "imagestreamTagNamespace")] = fmt.Sprintf("%v", rep.ClusterImage.ImageStreamTag.Namespace)
		}
		if err := p.client.Update(context.TODO(), pod); err != nil{
			log.Error(err,"failed to update pod with labels " + pod.Name + " in namespace " + pod. Namespace)
			labelErrors = append(labelErrors,err)
			continue
		}
	}

	if len(labelErrors) > 0{
		errMsg := ""
		for _, e := range labelErrors{
			errMsg+= e.Error() + " "
		}
		return errors.New(errMsg)
	}
	return nil
}