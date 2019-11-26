package validation

import (
	"fmt"
	"github.com/integr8ly/heimdall/pkg/domain"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"strconv"
	"strings"
	"time"
)

func ShouldCheck(meta v1.Object, images []*domain.ClusterImage) (bool, error) {
	minsBeforeCheckEnv := os.Getenv("HEIMDALL_RECHECK_MINS")
	var recheckAfter time.Duration = time.Duration(domain.MinRecheckIntervalMins)
	var parseErr error
	if minsBeforeCheckEnv != "" {
		v, err := strconv.ParseInt(minsBeforeCheckEnv, 10, 32)
		if err != nil {
			parseErr = &ParseErr{Message: fmt.Sprintf(" failed to parse env var to int HEIMDALL_RECHECK_MINS: %s  using default value %v ", err.Error(), recheckAfter)}
		} else {
			recheckAfter = time.Duration(v)
		}
	}
	annotations := meta.GetAnnotations()
	// If the images have changed we always want to check
	checked := strings.Split(annotations[domain.HeimdallImagesChecked], ",")
	for _, i := range images {
		found := false
		for _, c := range checked {
			if strings.TrimSpace(i.SHA256Path) == strings.TrimSpace(c) {
				found = true
				break
			}
		}
		if !found {
			return true, nil
		}
	}
	// if images havent changed we only want to check if enough time has passed
	lastChecked := annotations[domain.HeimdallLastChecked]
	if lastChecked == "" {
		return true, nil
	}
	checkedTime, err := time.Parse(time.RFC822Z, lastChecked)
	if err != nil {
		return false, &ParseErr{Message: err.Error()}
	}
	return checkedTime.Add(time.Minute * recheckAfter).Before(time.Now()), parseErr
}
