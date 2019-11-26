package registry

import (
	"fmt"
	"github.com/integr8ly/heimdall/pkg/domain"
	"github.com/integr8ly/heimdall/pkg/rhcc"
	"github.com/pkg/errors"
	"regexp"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("registry")

//go:generate moq -out ImageGetter_moq.go . ImageGetter
type ImageGetter interface {
	Get(string) (*domain.RemoteImageDigest, error)
}

//go:generate moq -out ImageVersionsGetter_moq.go . ImageVersionsGetter
type ImageVersionsGetter interface {
	AvailableTagsSortedByDate(string) ([]rhcc.Tag, error)
}

//go:generate moq -out ImageCVEGetter_moq.go . ImageCVEGetter
type ImageCVEGetter interface {
	CVES(org, tag string) ([]domain.CVE, error)
}

type ImageService struct {
	imageGetter    ImageGetter
	versionsGetter ImageVersionsGetter
	cveGetter      ImageCVEGetter
}

func NewImagesService(imageGetter ImageGetter, versGetter ImageVersionsGetter, cveGetter ImageCVEGetter) *ImageService {
	is := &ImageService{imageGetter: imageGetter, versionsGetter: versGetter, cveGetter: cveGetter}
	if is.imageGetter == nil {
		is.imageGetter = &Client{}
	}
	if is.versionsGetter == nil {
		is.versionsGetter = &rhcc.Client{}
	}
	if is.cveGetter == nil {
		is.cveGetter = &rhcc.Client{}
	}
	return is
}

func (i *ImageService) Check(image *domain.ClusterImage) (domain.ReportResult, error) {
	// get the registry image details based on the image we found
	fmt.Println("checking image : " + image.FullPath)
	result := domain.ReportResult{}
	result.ClusterImage = image
	clusterTagImage, err := i.imageGetter.Get(image.FullPath)
	if err != nil {
		return result, errors.Wrap(err, "failed to get image details from registry")
	}
	clusterImageSHA := image.GetSHAFromPath()
	if clusterImageSHA == "" {
		return result, errors.Wrap(err, "there was no sha found associated with this image")
	}
	// if not using image stream we need to get the sha for the sha
	if !image.FromImageStream {
		clusterImage, err := i.imageGetter.Get(image.SHA256Path)
		if err != nil {
			return result, errors.Wrap(err, "failed to get correct hash for image "+image.SHA256Path)
		}
		clusterImageSHA = clusterImage.Hash
	}
	tags, err := i.versionsGetter.AvailableTagsSortedByDate(image.OrgImagePath)
	if err != nil {
		return result, errors.Wrap(err, "failed to get available image tags")
	}
	majorMinorVersion := i.resolveMajorMinorVersion(tags, image.Tag)

	floatingTag, usingFloatingTag := i.resolveFloatingTag(tags, image.Tag)
	result.FloatingTag = floatingTag
	result.UsingFloatingTag = usingFloatingTag
	for j, t := range tags {

		mr := regexp.MustCompile("^v?" + majorMinorVersion + "(\\W)+")
		if !mr.MatchString(t.Name) {
			log.Info("skipping tag ", t.Name, " as it does not match on major minor patch version "+majorMinorVersion)
			continue
		}
		registryTagImage, err := i.imageGetter.Get(image.RegistryPath + ":" + t.Name)
		if err != nil {
			return result, errors.Wrap(err, "failed to get image details from registry for image "+image.RegistryPath+":"+t.Name)
		}
		// is the image on the cluster (ie the hash in the container) up to date with the has for the image tag in the registry
		result.UpToDateWithOwnTag = clusterTagImage.Hash == clusterImageSHA

		if registryTagImage.Hash == clusterImageSHA {
			if t.Name == "latest" && len(tags) > 1 {
				// we continue as there will be an actual specific image version that matches
				continue
			}
			result.CurrentVersion = t.Name
			result.CurrentGrade = t.FreshnessGrade

			floatingTagImage, err := i.imageGetter.Get(image.RegistryPath + ":" + result.FloatingTag)
			if err != nil {
				return result, errors.Wrap(err, "failed to get floating tag image from registry")
			}
			result.UpToDateWithFloatingTag = floatingTagImage.Hash == clusterImageSHA

			//check for latest patch version
			for i := 0; i <= j; i++ {
				match, err := regexp.MatchString("^v?"+majorMinorVersion+"(\\W)+", tags[i].Name)
				if err != nil {
					return result, errors.Wrap(err, "failed to match on regex ^"+majorMinorVersion)
				}
				if match {
					// they are in order so break out now
					result.LatestAvailablePatchVersion = tags[i].Name
					result.LatestGrade = tags[i].FreshnessGrade
					break
				}
			}
			break
		}
	}

	result.ActualImageRef = image.FullPath
	// upto date no need to check cves
	if result.LatestAvailablePatchVersion == result.CurrentVersion {
		return result, nil
	}
	resolvableCVEs, err := i.getResolvableCVEs(image, result.LatestAvailablePatchVersion, result.CurrentVersion)
	if err != nil {
		return result, err
	}
	result.ResolvableCVEs = resolvableCVEs
	return result, nil
}

func (i *ImageService) getResolvableCVEs(image *domain.ClusterImage, latestPatchVersion, currentVersion string) ([]domain.CVE, error) {
	latestImageCVEs, err := i.cveGetter.CVES(image.OrgImagePath, latestPatchVersion)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get CVEs affecting latest image tag "+latestPatchVersion)
	}
	currentImageCVEs, err := i.cveGetter.CVES(image.OrgImagePath, currentVersion)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get CVEs affecting current image tag "+currentVersion)
	}
	// assume the latest has the least cves
	var diff []domain.CVE
	// seems we can get the CVE more than once in the response
	checkCVE := map[string]struct{}{}
	for _, current := range currentImageCVEs {
		if _, ok := checkCVE[current.ID]; ok {
			continue
		}
		checkCVE[current.ID] = struct{}{}
		found := false
		for _, latest := range latestImageCVEs {
			if latest.ID == current.ID {
				found = true
				break
			}
		}
		if !found {
			diff = append(diff, current)
		}
	}
	return diff, nil
}

func (i *ImageService) resolveMajorMinorVersion(tags []rhcc.Tag, tag string) string {
	r := regexp.MustCompile("^v?\\d(\\d?\\.?)\\d?(\\d?\\d?\\d?)")
	if tag == "latest" {
		// as the tags are in date order we can use the tag right after latest as it will be floating and upto date with latest
		for i, t := range tags {
			if t.Name == "latest" {
				if len(tags)-1 > i {
					tag = tags[i+1].Name
				}
			}
		}
	}
	return r.FindString(tag)
}

func (i *ImageService) resolveFloatingTag(tags []rhcc.Tag, tag string) (string, bool) {
	majorMinor := i.resolveMajorMinorVersion(tags, tag)
	var floatingTag string
	if tag == "latest" {
		return tag, true
	}
	for _, t := range tags {
		// if we match the tag then we know if it is floating
		if t.Name == tag && t.Type == "floating" {
			return t.Name, true
		}
		if t.Name == majorMinor {
			floatingTag = t.Name
		}
	}
	// if not then it is not using the floating tag but a persistent tag
	if floatingTag != "" {
		return floatingTag, false
	}
	return "", false
}
