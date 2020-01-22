package cluster

import (
	"fmt"
	"github.com/integr8ly/heimdall/pkg/domain"
	v12 "github.com/openshift/api/apps/v1"
	v13 "github.com/openshift/api/image/v1"
	v14 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"regexp"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strings"
)

var log = logf.Log.WithName("image_service")
var replaceLocalImageRef = regexp.MustCompile("(^docker-.*)@")

type ImageService struct {
	client      kubernetes.Interface
	imageClient v14.ImageV1Interface
}

func NewImageService(k8s kubernetes.Interface, ic v14.ImageV1Interface) *ImageService {
	return &ImageService{
		client:      k8s,
		imageClient: ic,
	}
}

// if a deploymentconfig has triggers that use image change params this method will use those to find the underlying docker image no need to check all containers etc in this case
func (is *ImageService) FindImagesFromImageChangeParams(defaultNS string, params []*v12.DeploymentTriggerImageChangeParams, dcLabels map[string]string) ([]*domain.ClusterImage, error) {
	var images []*domain.ClusterImage
	var selectors []string
	// build a label selector based on the deploymentconfig labels
	for k, v := range dcLabels {
		selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
	}
	log.V(10).Info("selector ", "s ", strings.Join(selectors, ","))
	// find all the pods that match the dc labels
	pods, err := is.client.CoreV1().Pods(defaultNS).List(v1.ListOptions{LabelSelector: strings.Join(selectors, ",")})
	if err != nil {
		log.Error(err, "failed to list pods with labels "+strings.Join(selectors, ","))
		return nil, errors.Wrap(err, "failed to list pods with labels "+strings.Join(selectors, ","))
	}

	for _, p := range params {
		var ns = p.From.Namespace
		if ns == "" {
			ns = defaultNS
		}
		// we only care about image stream tags
		if p.From.Kind != "ImageStreamTag" {
			continue
		}
		// if the last triggered image is from the built in registry try find it on the ist
		ist, err := is.imageClient.ImageStreamTags(ns).Get(p.From.Name, v1.GetOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to find imagestreamtag "+ns+" "+p.From.Name)
		}
		actualImage, imageSHA, err := is.findImageAndSHABehindImageTag(ist)
		if err != nil {
			return nil, err
		}
		parsedImage := ParseImage(actualImage)
		parsedImage.FromImageStream = true
		parsedImage.ImageStreamTag = ist
		// if this is a local image ref we need to use the registry so as to avoid hitting the local registry
		if ist.Tag != nil && ist.Tag.ReferencePolicy.Type == v13.LocalTagReferencePolicy {
			found := replaceLocalImageRef.FindStringSubmatch(imageSHA)
			// will always be the match at index 1
			if len(found) == 2 {
				imageSHA = strings.Replace(imageSHA, found[1], parsedImage.RegistryPath, -1)
			}

		}
		parsedImage.SHA256Path = imageSHA
		parsedImage.Pods = []domain.PodAndContainerRef{}
		for _, p := range pods.Items {
			podContainerRef := domain.PodAndContainerRef{}
			podContainerRef.Namespace = p.Namespace
			podContainerRef.Name = p.Name
			podContainerRef.Containers = []string{}
			for _, c := range p.Spec.Containers {
				if c.Image == imageSHA {
					podContainerRef.Containers = append(podContainerRef.Containers, c.Name)
				}
			}
			parsedImage.Pods = append(parsedImage.Pods, podContainerRef)
		}
		images = append(images, parsedImage)
		continue
	}

	return images, nil
}

func (is *ImageService) findImageAndSHABehindImageTag(tag *v13.ImageStreamTag) (string, string, error) {
	if tag.Tag == nil {
		return "", "", errors.New("could not find image behind tag " + tag.Namespace + "/" + tag.Name + " as the tag was nil ")
	}
	if tag.Tag.From.Kind == "DockerImage" {
		return tag.Tag.From.Name, tag.Image.DockerImageReference, nil
	}
	baseImage := strings.Split(tag.Name, ":")[0]
	t, err := is.imageClient.ImageStreamTags(tag.Namespace).Get(baseImage+":"+tag.Tag.From.Name, v1.GetOptions{})
	if err != nil {
		return "", "", err
	}
	return is.findImageAndSHABehindImageTag(t)
}

// Finds images in pods with the specified labels (doesn't return images if they are in image streams)
func (is *ImageService) FindImagesFromLabels(ns string, deploymentLabels map[string]string) ([]*domain.ClusterImage, error) {
	var selectors []string
	var images []*domain.ClusterImage
	for k, v := range deploymentLabels {
		selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
	}
	log.V(10).Info("selector ", "s ", strings.Join(selectors, ","))
	pods, err := is.client.CoreV1().Pods(ns).List(v1.ListOptions{LabelSelector: strings.Join(selectors, ",")})
	if err != nil {
		log.Error(err, "failed to list pods with labels "+strings.Join(selectors, ","))
		return nil, errors.Wrap(err, "failed to list pods with labels "+strings.Join(selectors, ","))
	}
	// create a unique set of images
	imageIDS := map[string]*domain.ClusterImage{}
	// get all images from pods
	for _, p := range pods.Items {
		pcr := domain.PodAndContainerRef{
			Name:       p.Name,
			Namespace:  p.Namespace,
			Containers: []string{},
		}
		for _, cs := range p.Status.ContainerStatuses {
			//check if this image is coming from the the internal registry if so skip
			if strings.Contains(cs.ImageID, "docker-registry") {
				log.Info("skipping image ", cs.ImageID, " as it is internal. It will be picked up as part of image stream checking ")
				continue
			}
			imageID := strings.Replace(cs.ImageID, "docker-pullable://", "", 1)

			// check have we looked at this image already. Can happen when multiple pods or multiple containers with same image
			var image *domain.ClusterImage
			if i, ok := imageIDS[imageID]; ok {
				image = i
			} else {
				image = ParseImage(cs.Image)
				image.SHA256Path = imageID

			}

			for _, c := range p.Spec.Containers {
				pcr.Containers = append(pcr.Containers, c.Name)
			}

			foundPod := false
			for _, sp := range image.Pods {
				if sp.Name == p.Name {
					foundPod = true
					break
				}
			}
			if !foundPod {
				image.Pods = append(image.Pods, pcr)
			}
			if _, ok := imageIDS[imageID]; !ok {
				imageIDS[imageID] = image
				images = append(images, image)
			}
		}
	}
	return images, nil
}

// parse image it will look like
func ParseImage(i string) *domain.ClusterImage {
	reg := regexp.MustCompile("@?sha256")
	si := reg.ReplaceAllString(i, "")

	registryURLParts := strings.Split(si, "/")
	imageParts := strings.Split(registryURLParts[len(registryURLParts)-1], ":")
	registryURL := strings.Join(registryURLParts[:len(registryURLParts)-1], "/") + "/" + imageParts[0]
	imagePath := strings.Split(strings.Split(si, ":")[0], "/")
	if len(imageParts) == 1 {
		imageParts = append(imageParts, "latest")
	}
	return &domain.ClusterImage{
		FullPath:     i,
		OrgImagePath: imagePath[1] + "/" + imagePath[2],
		ImageName:    imageParts[0],
		Tag:          imageParts[1],
		RegistryPath: registryURL,
		Org:          registryURLParts[1],
	}
}
