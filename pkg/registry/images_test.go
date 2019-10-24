package registry_test

import (
	"github.com/integr8ly/heimdall/pkg/cluster"
	"github.com/integr8ly/heimdall/pkg/domain"
	"github.com/integr8ly/heimdall/pkg/registry"
	"github.com/integr8ly/heimdall/pkg/rhcc"
	"strings"
	"testing"
)

func TestImageService_Check(t *testing.T) {
	cases := []struct{
		Name string
		Image string
		SHAImage string
		ImageStream bool
		ImageGetter func()registry.ImageGetter
		CVEGetter func()registry.ImageCVEGetter
		VersionGetter func()registry.ImageVersionsGetter
		Validate func(t *testing.T, result *domain.ReportResult)
		ExpectError bool
	}{
		{
			Name: "test check returns CVEs and later version when digest does not match",
			Image: "registry.redhat.io/amq7/amq-online-1-api-server:1.0.0",
			SHAImage: "registry.redhat.io/amq7/amq-online-1-api-server@sha256:someotherhash",
			ImageStream:true,
			ImageGetter: func() registry.ImageGetter {
				return &registry.ImageGetterMock{
					GetFunc: func(in1 string) (digest *domain.RemoteImageDigest, e error) {
						if strings.Contains(in1, "1.0.1") || strings.Contains(in1, "1.0.2") {
							return &domain.RemoteImageDigest{Hash: "somehash", Algorithm: "sha256"}, nil
						}
						return &domain.RemoteImageDigest{Hash: "someotherhash", Algorithm: "sha256"}, nil
					},
				}
			},
			VersionGetter: func() registry.ImageVersionsGetter {
				return &registry.ImageVersionsGetterMock{
					AvailableTagsSortedByDateFunc: func(in1 string) (strings []rhcc.Tag, e error) {
						return []rhcc.Tag{{Name:"1.0",Added:"20191124T09:53:00.000-0500", TimeAdded:0,Type:"floating"},{Name:"1.0.1",Added:"20191125T09:53:00.000-0500", TimeAdded:1, Type:"persistent"}, {Name:"1.0.2", Added:"20191126T09:53:00.000-0500", TimeAdded:2, Type:"persistent"}}, nil
					},
				}
			},
			CVEGetter: func() registry.ImageCVEGetter {
				return &registry.ImageCVEGetterMock{
					CVESFunc: func(org string, tag string) (cves []domain.CVE, e error) {
						if tag == "1.0"{
							return []domain.CVE{{
								Severity:   "minor",
								ID:         "1",
								AdvisoryID: "1",
							},
								{
									Severity:   "important",
									ID:         "2",
									AdvisoryID: "2",
								}}, nil
						}
						if tag == "1.0.2"{
							return []domain.CVE{{
								Severity:   "minor",
								ID:         "1",
								AdvisoryID: "1",
							},
							}, nil
						}
						return nil, nil
					},
				}
			},
			Validate: func(t *testing.T, res *domain.ReportResult){
				if res.CurrentVersion != "1.0"{
					t.Fatal("expected version 1.0.0 but got ",res.CurrentVersion)
				}
				if res.LatestAvailablePatchVersion != "1.0.2"{
					t.Fatal("expected the latest available version to be 1.0.2 but got ", res.LatestAvailablePatchVersion)
				}
				if len(res.ResolvableCVEs) != 1{
					t.Fatal("expected the resolvable CVEs to be  ", res.ResolvableCVEs)
				}
			},
		},
	}

	for _, tc := range cases{
		t.Run(tc.Name, func(t *testing.T) {
			is := registry.NewImagesService(tc.ImageGetter(), tc.VersionGetter(), tc.CVEGetter() )
			img := cluster.ParseImage(tc.Image)
			img.SHA256Path = tc.SHAImage
			img.FromImageStream = tc.ImageStream
			result, err := is.Check(img)
			if tc.ExpectError && err == nil{
				t.Fatal("expected an error but did not get one")
			}
			if ! tc.ExpectError && err != nil{
				t.Fatal("did not expect an error but got one ", err)
			}
			if tc.Validate != nil{
				tc.Validate(t, &result)
			}
		})
	}
}
