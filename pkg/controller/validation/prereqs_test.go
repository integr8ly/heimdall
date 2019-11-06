package validation_test

import (
	"github.com/integr8ly/heimdall/pkg/controller/validation"
	"github.com/integr8ly/heimdall/pkg/domain"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func TestShouldCheck(t *testing.T) {
	cases := []struct{
		Name string
		Object v1.Object
		Expect bool
		ExpectErr bool
	}{
		{
			Name :"test should not check when not labeled by heimdall",
			Object:&v12.Deployment{
				ObjectMeta:v1.ObjectMeta{
					Annotations: map[string]string{},
					Labels: map[string]string{},
				},
			},
			Expect: false,
			ExpectErr: false,
		},
		{
			Name :"test should not check when labeled by heimdall and not enough time has passed",
			Object:&v12.Deployment{
				ObjectMeta:v1.ObjectMeta{
					Annotations: map[string]string{
						domain.HeimdallLastChecked: time.Now().Format(domain.TimeFormat),
					},
					Labels: map[string]string{
						domain.HeimdallMonitored:"true",
					},
				},
			},
			Expect: false,
			ExpectErr: false,
		},
		{
			Name :"test should check when labeled by heimdall and correct time has passed",
			Object:&v12.Deployment{
				ObjectMeta:v1.ObjectMeta{
					Annotations: map[string]string{
						domain.HeimdallLastChecked: time.Now().AddDate(0,0, -1).Format(domain.TimeFormat),
					},
					Labels: map[string]string{
						domain.HeimdallMonitored:"true",
					},
				},
			},
			Expect: true,
			ExpectErr: false,
		},
	}

	for _, tc := range cases{
		t.Run(tc.Name, func(t *testing.T) {
			should, err := validation.ShouldCheck(tc.Object)
			if tc.ExpectErr && err == nil{
				t.Fatal("expected and error but got none")
			}
			if !tc.ExpectErr && err != nil{
				t.Fatal("did not expect an error but got one ", err)
			}
			if tc.Expect != should{
				t.Fatal("expected ", tc.Expect, " but got ", should)
			}
		})
	}
}
