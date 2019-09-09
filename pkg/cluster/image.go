package cluster

import (
	"fmt"
	apps "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	is "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"regexp"
	"strings"
)

type ImageService struct {
	k8sclient kubernetes.Interface
	dcClient  *apps.AppsV1Client
	isClient  *is.ImageV1Client
}

var parseSha = regexp.MustCompile("sha256:.*$")

func NewImageService(k8sclient kubernetes.Interface, dc *apps.AppsV1Client, ic *is.ImageV1Client) *ImageService {
	return &ImageService{k8sclient: k8sclient, dcClient: dc, isClient: ic}
}

// need to deal with deployment configs
func (is *ImageService) FindImages(ns string, regex *regexp.Regexp) ([]*Image, error) {
	images, err := is.findImagesInDeployment(ns, regex)
	if err != nil {
		return images, err
	}
	dcImages, err := is.findImagesInDeploymentConfigs(ns, regex)
	if err != nil {
		return images, err
	}
	images = append(images, dcImages...)

	return images, nil
}

func (is *ImageService) findImagesInDeployment(ns string, regex *regexp.Regexp) ([]*Image, error) {
	var images []*Image
	deployments, err := is.k8sclient.AppsV1().Deployments(ns).List(v1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list deployments in ns "+ns)
	}
	for _, d := range deployments.Items {
		if !regex.MatchString(d.Name) {
			log.Println("not checking ", d.Name, "does not match regex")
			continue
		}
		// for each container there will be a pod. Find the pod and get the imageID
		var selectors []string
		for k, v := range d.Spec.Template.Labels {
			selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
		}
		pl, err := is.k8sclient.CoreV1().Pods(ns).List(v1.ListOptions{LabelSelector: strings.Join(selectors, ",")})
		if err != nil {
			return images, err
		}

		containers := d.Spec.Template.Spec.Containers
		for _, c := range containers {
			for _, p := range pl.Items {
				for _, pc := range p.Status.ContainerStatuses {
					log.Println("deployment container name ", c.Name, "pod container name ", pc.Name)
					if c.Name == pc.Name {
						i := parseImage(pc.Image)
						match := parseSha.FindString(pc.ImageID)
						if match == "" {
							return nil, errors.New("failed to reliably parse the image sha ")
						}
						i.SHA = "@" + match
						fmt.Println("image details", i.ImageName, i.Tag, "sha", i.SHA)
						images = append(images, i)
					}
				}
			}
		}
	}

	return images, nil
}

func (is *ImageService) findImagesInDeploymentConfigs(ns string, regex *regexp.Regexp) ([]*Image, error) {
	var images []*Image
	dcs, err := is.dcClient.DeploymentConfigs(ns).List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, dc := range dcs.Items {
		if !regex.MatchString(dc.Name) {
			continue
		}
		fmt.Println("checking ", dc.Name)
		for _, trigger := range dc.Spec.Triggers {
			// IF there is an imagestream trigger we check the image stream tag. Get the from image (IE the one in the registry) and add that to our images to check

			// for each trigger find the container it is for and get the img from that container
			// for each img we get check the image stream tag, ensure the deployment is upto date with the image stream tag, ensure the image stream tag is upto date with the latest registry image it points at.

			if trigger.ImageChangeParams != nil && trigger.ImageChangeParams.From.Kind == "ImageStreamTag" {
				fmt.Println("image change trigger", trigger.ImageChangeParams.From.Name)
				isTag, err := is.isClient.ImageStreamTags(trigger.ImageChangeParams.From.Namespace).Get(trigger.ImageChangeParams.From.Name, v1.GetOptions{})
				if err != nil{
					return nil , err
				}

				for _, c := range dc.Spec.Template.Spec.Containers {
					if containsString(trigger.ImageChangeParams.ContainerNames, c.Name) && !trigger.ImageChangeParams.Automatic {
						// check the container img matches the imagestream tag
						if c.Image != isTag.Image.ObjectMeta.Name {
							// this means the the dc needs to be rolled out
							log.Println("dc needs to be rolled out image is not upto date with image stream tag")
						}
					}
				}
				if isTag != nil && isTag.Tag != nil  {
					fmt.Println("image in imagestream tag ", isTag.Tag.From.Name)
				}
				registryImg, imageSHA, err := is.findRegistryImageAndSHAFromImageStreamTag(trigger.ImageChangeParams.From.Namespace, trigger.ImageChangeParams.From.Name)
				if err != nil {
					return images, err
				}
				i := parseImage(registryImg)
				match := parseSha.FindString(imageSHA)
				if match == "" {
					return nil, errors.New("failed to reliably parse the image sha ")
				}

				i.SHA = "@" + match

				images = append(images, i)
				continue
			}
			// TODO
			// handle no image stream in a deployment config
		}
	}
	// get deployment configs
	// look for imagestream triggers
	// if found find image stream and tag
	// find the image in the registry associated with that image and tag
	// check if the current hash matches that latest hash for that image and tag
	return images, nil
}

func (is *ImageService) findRegistryImageAndSHAFromImageStreamTag(ns, name string) (string, string, error) {
	isTag, err := is.isClient.ImageStreamTags(ns).Get(name, v1.GetOptions{})
	if err != nil {
		return "", "", err
	}

	if isTag.Tag != nil && isTag.Tag.From.Kind == "ImageStreamTag" {

		if isTag.Tag.From.Namespace != "" {
			ns = isTag.Tag.From.Namespace
		}
		imageName := strings.Split(isTag.Name, ":")[0]
		fmt.Println("image stream image ", imageName, isTag.Tag.From.Name)
		return is.findRegistryImageAndSHAFromImageStreamTag(ns, imageName+":"+isTag.Tag.From.Name)
	}
	if isTag.Tag == nil{
		return isTag.Image.DockerImageReference, isTag.Image.Name, nil
	}
	return isTag.Tag.From.Name, isTag.Image.ObjectMeta.Name, nil
}

func containsString(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}


func parseImage(i string) *Image {
	fmt.Println("parsing image ", i)

	registryURLParts := strings.Split(i, "/")
	imageParts := strings.Split(registryURLParts[len(registryURLParts)-1], ":")
	if len(imageParts) == 1 {
		imageParts = append(imageParts, "latest")
	}
	fmt.Println("reg parts", registryURLParts)
	return &Image{
		RegistryPath: strings.Split(i, ":")[0],
		ImageName:    imageParts[0],
		Tag:          imageParts[1],
		Registry:     registryURLParts[0],
		OrgPath:      registryURLParts[len(registryURLParts)-2] + "/" + imageParts[0],
	}
}

type Image struct {
	RegistryPath string
	OrgPath      string
	Tag          string
	ImageName    string
	SHA          string
	Registry     string
}
