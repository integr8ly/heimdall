package cluster_test

import (
	"github.com/integr8ly/heimdall/pkg/cluster"
	"github.com/integr8ly/heimdall/pkg/domain"
	v1 "github.com/openshift/api/apps/v1"
	v12 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	"k8s.io/client-go/kubernetes"
	"reflect"
	"testing"
)

func TestParseImage(t *testing.T) {
	cases := []struct{
		Name string
		Image string
		Expect *domain.ClusterImage
	}{
		{
			Name:"test parsing image with sha",
			Image: "registry.redhat.io/3scale-amp26/system:eb98e41a76f7ed3d7dd81a3687dcb0452b8c414a0ef80966afcfcc00b1c5accb",
			Expect:&domain.ClusterImage{
				FullPath:     "registry.redhat.io/3scale-amp26/system:eb98e41a76f7ed3d7dd81a3687dcb0452b8c414a0ef80966afcfcc00b1c5accb",
				OrgImagePath: "3scale-amp26/system",
				Tag:          "eb98e41a76f7ed3d7dd81a3687dcb0452b8c414a0ef80966afcfcc00b1c5accb",
				ImageName:    "system",
				RegistryPath: "registry.redhat.io/3scale-amp26/system",
				Org:          "3scale-amp26",
				SHA256Path:   "",
			},
		},
		{
			Name:"test parsing image with tag",
			Image: "registry.access.redhat.com/jboss-amq-6/amq63-openshift:1.3",
			Expect:&domain.ClusterImage{
				FullPath:     "registry.access.redhat.com/jboss-amq-6/amq63-openshift:1.3",
				OrgImagePath: "jboss-amq-6/amq63-openshift",
				Tag:          "1.3",
				ImageName:    "amq63-openshift",
				RegistryPath: "registry.access.redhat.com/jboss-amq-6/amq63-openshift",
				Org:          "jboss-amq-6",
				SHA256Path:   "",
			},
		},
	}

	for _, tc := range cases{
		t.Run(tc.Name, func(t *testing.T) {
			ci := cluster.ParseImage(tc.Image)
			if !reflect.DeepEqual(*ci, *tc.Expect){
				t.Fatal("expected ", tc.Expect, " but got ", ci)
			}
		})
	}
}

func TestImageService_FindImagesFromImageChangeParams(t *testing.T) {
	cases := []struct{
		Name string
		Namespace string
		ChangeParams []*v1.DeploymentTriggerImageChangeParams
		DeploymentLabels map[string]string
		K8sClient kubernetes.Interface
		ImageClient v12.ImageV1Interface
		ExpectErr bool
		Validate func(t *testing.T, images []*domain.ClusterImage)
	}{
		{
			Name:"test finding images ",

		},
	}

	for _, tc := range cases{
		t.Run(tc.Name, func(t *testing.T) {
			is := cluster.NewImageService(tc.K8sClient, tc.ImageClient)
			images, err := is.FindImagesFromImageChangeParams(tc.Namespace,tc.ChangeParams, tc.DeploymentLabels)
			if tc.ExpectErr && err == nil{
				t.Fatal("expected an error but got none")
			}
			if ! tc.ExpectErr && err != nil{
				t.Fatal("did not expect an error but got one ", err)
			}
			if tc.Validate != nil{
				tc.Validate(t, images)
			}
		})
	}
}

func TestImageService_FindImagesFromLabels(t *testing.T) {

}
