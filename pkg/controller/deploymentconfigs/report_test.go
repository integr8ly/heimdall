package deploymentconfigs_test

import (
	"github.com/integr8ly/heimdall/pkg/controller/deployments"
	"testing"
)

func TestReports_Generate(t *testing.T) {
	cases := []struct{
		Name string
	}{
		{

		},
	}

	for _,tc := range cases{
		t.Run(tc.Name, func(t *testing.T) {
			reporter := deployments.NewReport()
		})
	}
}
