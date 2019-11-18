package validation

import (
	"github.com/integr8ly/heimdall/pkg/domain"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func ShouldCheck(meta v1.Object) (bool, error) {
	annotations := meta.GetAnnotations()
	labels := meta.GetLabels()
	if _, ok := labels[domain.HeimdallMonitored]; !ok {
		return false, nil
	}
	lastChecked := annotations[domain.HeimdallLastChecked]
	if lastChecked == "" {
		return true, nil
	}
	checked, err := time.Parse(time.RFC822Z, lastChecked)
	if err != nil {
		return false, &ParseErr{Message: err.Error()}
	}
	return checked.Add(time.Minute * 15).Before(time.Now()), nil

}
