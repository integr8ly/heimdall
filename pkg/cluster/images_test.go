package cluster_test

import (
	"fmt"
	"github.com/integr8ly/heimdall/pkg/cluster"
	"github.com/integr8ly/heimdall/pkg/domain"
	v1 "github.com/openshift/api/apps/v1"
	v15 "github.com/openshift/api/image/v1"
	v12 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	v12fake "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1/fake"
	"github.com/pkg/errors"
	v13 "k8s.io/api/core/v1"
	v14 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	testing2 "k8s.io/client-go/testing"
	"reflect"
	"testing"
)

func TestParseImage(t *testing.T) {
	cases := []struct {
		Name   string
		Image  string
		Expect *domain.ClusterImage
	}{
		{
			Name:  "test parsing image with sha",
			Image: "registry.redhat.io/3scale-amp26/system:eb98e41a76f7ed3d7dd81a3687dcb0452b8c414a0ef80966afcfcc00b1c5accb",
			Expect: &domain.ClusterImage{
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
			Name:  "test parsing image with tag",
			Image: "registry.access.redhat.com/jboss-amq-6/amq63-openshift:1.3",
			Expect: &domain.ClusterImage{
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

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			ci := cluster.ParseImage(tc.Image)
			if !reflect.DeepEqual(*ci, *tc.Expect) {
				t.Fatal("expected ", tc.Expect, " but got ", ci)
			}
		})
	}
}

type podArgs struct {
	Name    string
	NS      string
	Image   string
	ImageID string
}

func buildPodList(podArgs []podArgs) *v13.PodList {
	pl := &v13.PodList{
		Items: []v13.Pod{},
	}

	for _, pa := range podArgs {
		pl.Items = append(pl.Items, v13.Pod{
			TypeMeta: v14.TypeMeta{},
			ObjectMeta: v14.ObjectMeta{
				Namespace: pa.NS,
				Name:      pa.Name,
			},
			Spec: v13.PodSpec{
				Containers: []v13.Container{
					{
						Image: pa.Image,
					},
				},
			},
			Status: v13.PodStatus{
				ContainerStatuses: []v13.ContainerStatus{
					{
						Image:   pa.Image,
						ImageID: pa.ImageID,
					},
				},
			},
		})
	}
	return pl
}

func TestImageService_FindImagesFromImageChangeParams(t *testing.T) {
	var testImageRef = "registry.redhat.io/something/componenta@sha256:ca34c10e89826985036b8d7316f0b5177be1da1aef9997b4cd2c30dd87ba4ca0"
	cases := []struct {
		Name             string
		Namespace        string
		ChangeParams     []*v1.DeploymentTriggerImageChangeParams
		DeploymentLabels map[string]string
		K8sClient        func() kubernetes.Interface
		ImageClient      func() v12.ImageV1Interface
		ExpectErr        bool
		Validate         func(t *testing.T, images []*domain.ClusterImage)
	}{
		{
			Name:      "test finding images from image stream with single pod ",
			Namespace: "test",
			ChangeParams: []*v1.DeploymentTriggerImageChangeParams{
				&v1.DeploymentTriggerImageChangeParams{
					Automatic:      false,
					ContainerNames: []string{""},
					From: v13.ObjectReference{
						Kind:      "ImageStreamTag",
						Namespace: "openshift",
						Name:      "test",
					},
					LastTriggeredImage: "",
				},
			},
			K8sClient: func() kubernetes.Interface {
				c := &fake.Clientset{}
				c.AddReactor("list", "pods", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {
					return true, buildPodList([]podArgs{{
						Name:  "test-pod",
						Image: testImageRef,
						NS:    "test",
					}}), nil
				})
				return c
			},
			ImageClient: func() v12.ImageV1Interface {
				fc := &v12fake.FakeImageV1{}
				fc.Fake = &testing2.Fake{}
				fc.Fake.AddReactor("get", "imagestreamtags", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {
					return true, &v15.ImageStreamTag{
						ObjectMeta: v14.ObjectMeta{
							Name:      "test",
							Namespace: "openshift",
						},
						Image: v15.Image{
							DockerImageReference: testImageRef,
						},
						Tag: &v15.TagReference{
							From: &v13.ObjectReference{
								Kind: "DockerImage",
								Name: "registry.redhat.io/something/component:2.0",
							},
						},
					}, nil
				})
				return fc
			},
			Validate: func(t *testing.T, images []*domain.ClusterImage) {
				if len(images) != 1 {
					t.Fatalf("expected 1 cluster image but got %v", len(images))
				}
				image := images[0]
				if image.SHA256Path != testImageRef {
					t.Fatal("expected a sha path of " + testImageRef +
						"but got " + images[0].SHA256Path)
				}
				if image.Tag != "2.0" {
					t.Fatal("expected the image tag to be 2.0 but got ", image.Tag)
				}
				if len(image.Pods) != 1 {
					t.Fatalf("expected a single pod reference but got %v ", len(image.Pods))
				}
				if image.Pods[0].Name != "test-pod" && image.Pods[0].Namespace != "test" {
					t.Fatal("expected a pod with name test-pod and namespace test got ", image.Pods[0].Name, image.Pods[0].Namespace)
				}
			},
		},
		{
			Name:      "test finding images from image stream with multiple pods ",
			Namespace: "test",
			ChangeParams: []*v1.DeploymentTriggerImageChangeParams{
				&v1.DeploymentTriggerImageChangeParams{
					Automatic:      false,
					ContainerNames: []string{""},
					From: v13.ObjectReference{
						Kind:      "ImageStreamTag",
						Namespace: "test1",
						Name:      "test1",
					},
					LastTriggeredImage: "",
				},
				&v1.DeploymentTriggerImageChangeParams{
					Automatic:      false,
					ContainerNames: []string{""},
					From: v13.ObjectReference{
						Kind:      "ImageStreamTag",
						Namespace: "test2",
						Name:      "test2",
					},
					LastTriggeredImage: "",
				},
			},
			K8sClient: func() kubernetes.Interface {
				c := &fake.Clientset{}
				c.AddReactor("list", "pods", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {
					pl := buildPodList([]podArgs{{
						NS:    "test",
						Name:  "test-pod",
						Image: testImageRef,
					}, {
						NS:    "test",
						Name:  "test-pod2",
						Image: testImageRef,
					}})
					pl.Items[0].Spec.Containers = append(pl.Items[0].Spec.Containers, v13.Container{})
					return true, pl, nil
				})
				return c
			},
			ImageClient: func() v12.ImageV1Interface {
				fc := &v12fake.FakeImageV1{}
				fc.Fake = &testing2.Fake{}
				fc.Fake.AddReactor("get", "imagestreamtags", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {
					getAct := action.(testing2.GetAction)
					if getAct.GetName() == "test1" {
						return true, &v15.ImageStreamTag{
							ObjectMeta: v14.ObjectMeta{
								Name:      "test",
								Namespace: action.GetNamespace(),
							},
							Image: v15.Image{
								DockerImageReference: testImageRef,
							},
							Tag: &v15.TagReference{
								From: &v13.ObjectReference{
									Kind: "DockerImage",
									Name: "registry.redhat.io/something/component:2.0",
								},
							},
						}, nil
					}
					if getAct.GetName() == "test2" {
						return true, &v15.ImageStreamTag{
							ObjectMeta: v14.ObjectMeta{
								Name:      "test2",
								Namespace: action.GetNamespace(),
							},
							Image: v15.Image{
								DockerImageReference: testImageRef,
							},
							Tag: &v15.TagReference{
								From: &v13.ObjectReference{
									Kind: "DockerImage",
									Name: "registry.redhat.io/something/component:2.0",
								},
							},
						}, nil
					}
					return true, nil, errors.New("not found")
				})
				return fc
			},
			Validate: func(t *testing.T, images []*domain.ClusterImage) {
				if len(images) != 2 {
					t.Fatalf("expected 1 cluster image but got %v", len(images))
				}
				// for the test we use the same image twice
				for _, image := range images {
					if image.SHA256Path != testImageRef {
						t.Fatal("expected a sha path of " + testImageRef +
							"but got " + images[0].SHA256Path)
					}
					if image.Tag != "2.0" {
						t.Fatal("expected the image tag to be 2.0 but got ", image.Tag)
					}
					if len(image.Pods) != 2 {
						t.Fatalf("expected a two pod references but got %v ", len(image.Pods))
					}
					if image.Pods[0].Name != "test-pod" && image.Pods[0].Namespace != "test" {
						t.Fatal("expected a pod with name test-pod and namespace test got ", image.Pods[0].Name, image.Pods[0].Namespace)
					}
					if image.Pods[0].Name != "test-pod2" && image.Pods[0].Namespace != "test" {
						t.Fatal("expected a pod with name test-pod2 and namespace test got ", image.Pods[0].Name, image.Pods[0].Namespace)
					}
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			is := cluster.NewImageService(tc.K8sClient(), tc.ImageClient())
			images, err := is.FindImagesFromImageChangeParams(tc.Namespace, tc.ChangeParams, tc.DeploymentLabels)
			if tc.ExpectErr && err == nil {
				t.Fatal("expected an error but got none")
			}
			if !tc.ExpectErr && err != nil {
				t.Fatal("did not expect an error but got one ", err)
			}
			if tc.Validate != nil {
				tc.Validate(t, images)
			}
		})
	}
}

func TestImageService_FindImagesFromLabels(t *testing.T) {
	var testImageID = "docker-pullable://registry.redhat.io/amq7/amq-online-1-address-space-controller@sha256:%v"
	var testImageSha = "registry.redhat.io/amq7/amq-online-1-address-space-controller@sha256:%v"
	var testImage = "registry.redhat.io/amq7/amq-online-1-address-space-controller:%v"
	cases := []struct {
		Name             string
		Namespace        string
		Labels           map[string]string
		DeploymentLabels map[string]string
		K8sClient        func() kubernetes.Interface
		ImageClient      func() v12.ImageV1Interface
		ExpectErr        bool
		Validate         func(t *testing.T, images []*domain.ClusterImage)
	}{
		{
			Name:      "Test find images in pods",
			Namespace: "test",
			Labels:    map[string]string{},
			K8sClient: func() kubernetes.Interface {
				c := &fake.Clientset{}
				c.AddReactor("list", "pods", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {
					pl := buildPodList([]podArgs{{
						NS:      "test",
						Name:    "test-pod",
						Image:   fmt.Sprintf(testImage, "0"),
						ImageID: fmt.Sprintf(testImageID, "test0sha"),
					}, {
						NS:      "test",
						Name:    "test-pod2",
						Image:   fmt.Sprintf(testImage, "1"),
						ImageID: fmt.Sprintf(testImageID, "test1sha"),
					}})
					return true, pl, nil
				})
				return c
			},
			ImageClient: func() v12.ImageV1Interface {
				return nil
			},
			ExpectErr: false,
			Validate: func(t *testing.T, images []*domain.ClusterImage) {
				if len(images) != 2 {
					t.Fatalf("expected 1 cluster images but got %v", len(images))
				}
				for i, image := range images {
					if image.SHA256Path != fmt.Sprintf(testImageSha, fmt.Sprintf("test%vsha", i)) {
						t.Fatal("expected sha path to be " + fmt.Sprintf(testImageSha, fmt.Sprintf("test%vsha", i)) + " but got " + image.SHA256Path)
					}
					if image.FullPath != fmt.Sprintf(testImage, i) {
						t.Fatal("expected the full path to be " + fmt.Sprintf(testImage, i) + " but got " + image.FullPath)
					}
				}
			},
		},
		{
			Name:      "Test find multiple images in multiple pods",
			Namespace: "test",
			Labels:    map[string]string{},
			K8sClient: func() kubernetes.Interface {
				c := &fake.Clientset{}
				c.AddReactor("list", "pods", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {
					pl := buildPodList([]podArgs{{
						NS:      "test",
						Name:    "test-pod",
						Image:   testImage,
						ImageID: testImageID,
					}, {
						NS:      "test",
						Name:    "test-pod2",
						Image:   testImage,
						ImageID: testImageID,
					}})
					return true, pl, nil
				})
				return c
			},
			ImageClient: func() v12.ImageV1Interface {
				return nil
			},
			ExpectErr: false,
			Validate: func(t *testing.T, images []*domain.ClusterImage) {
				if len(images) != 1 {
					t.Fatalf("expected 1 cluster images but got %v", len(images))
				}

				for _, i := range images {
					// Expect 2 pods to be returned as they both have the same image and we want to label them all
					if len(i.Pods) != 2 {
						t.Fatal("expected 1 pod to be returned")
					}
					if i.SHA256Path != testImageSha {
						t.Fatal("expected sha path to be " + testImageSha + " but got " + i.SHA256Path)
					}
					if i.FullPath != testImage {
						t.Fatal("expected the full path to be " + testImage + " but got " + i.FullPath)
					}
				}
			},
		},
		{
			Name:      "Expect error when fail to get pods",
			Namespace: "test",
			Labels:    map[string]string{},
			K8sClient: func() kubernetes.Interface {
				c := &fake.Clientset{}
				c.AddReactor("list", "pods", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("failed to get pods")
				})
				return c
			},
			ImageClient: func() v12.ImageV1Interface {
				return nil
			},
			ExpectErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			is := cluster.NewImageService(tc.K8sClient(), tc.ImageClient())
			clusterImages, err := is.FindImagesFromLabels(tc.Namespace, tc.Labels)
			if tc.ExpectErr && err == nil {
				t.Fatal("expected an error but got none")
			}
			if !tc.ExpectErr && err != nil {
				t.Fatal("did not expect an error but got one ", err)
			}
			if tc.Validate != nil {
				tc.Validate(t, clusterImages)
			}
		})

	}
}
