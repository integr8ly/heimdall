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

type registryDigest struct {
	TagDigest string
	SHADigest string
}

func (i *ImageService) clusterImageRegistryDigests(image *domain.ClusterImage) (registryDigest, error) {

	var (
		clusterImageSHAHash, clusterImageTagHash string
		err                                      error
	)
	// Even though we have a SHA, if it is not from an imagestream the digest in the container seems to be calculated differently. However when we ask
	// the registry for the image that matches the digest in the container, it gives us back an image and that digest will match the tag being used.
	if !image.FromImageStream {
		clusterSHAImage, err := i.imageGetter.Get(image.SHA256Path)
		if err != nil {
			return registryDigest{}, errors.Wrap(err, "failed to get correct hash for image "+image.SHA256Path)
		}
		clusterImageSHAHash = clusterSHAImage.Hash
	} else {
		clusterImageSHAHash = image.GetSHAFromPath()
	}

	clusterTagImage, err := i.imageGetter.Get(image.FullPath)
	if err != nil {
		return registryDigest{}, errors.Wrap(err, "failed to get image details from registry")
	}
	clusterImageTagHash = clusterTagImage.Hash

	return registryDigest{
		TagDigest: clusterImageTagHash,
		SHADigest: clusterImageSHAHash,
	}, nil
}

// Check takes the cluster image and runs through a set of checks to find out whether the image in the cluster is
// up to date with the image in the registry, whether there is a new patch image available and also figures out which
// CVEs would be fixed by updating.
func (i *ImageService) Check(image *domain.ClusterImage) (domain.ReportResult, error) {
	// get the registry image details based on the image we found
	fmt.Println("checking image : " + image.FullPath)
	result := domain.ReportResult{}
	result.ClusterImage = image
	clusterImageDigests, err := i.clusterImageRegistryDigests(image)
	if err != nil {
		return result, errors.Wrap(err, " failed to disover the cluster image SHA ")
	}
	tags, err := i.versionsGetter.AvailableTagsSortedByDate(image.OrgImagePath)
	if err != nil {
		return result, errors.Wrap(err, "failed to get available image tags")
	}
	majorMinorVersion := i.resolveMajorMinorVersion(tags, image.Tag)
	isPersistent, index := i.isPersistentTag(tags, image.Tag)
	floatingTag, usingFloatingTag := i.resolveFloatingTag(tags, image.Tag)
	result.FloatingTag = floatingTag
	result.UsingFloatingTag = usingFloatingTag
	result.ActualImageRef = image.FullPath
	result.UpToDateWithOwnTag = clusterImageDigests.TagDigest == clusterImageDigests.SHADigest
	floatingTagImage, err := i.imageGetter.Get(image.RegistryPath + ":" + result.FloatingTag)
	if err != nil {
		return result, errors.Wrap(err, "failed to get floating tag image from registry")
	}
	if isPersistent {
		t := tags[index]
		result.CurrentVersion = t.Name
		result.CurrentGrade = t.FreshnessGrade
		result.UpToDateWithFloatingTag = floatingTagImage.Hash == clusterImageDigests.SHADigest
		//check for latest patch version
		nextTag, err := findNextPatchImage(tags[:index+1], majorMinorVersion)
		if err != nil {
			return result, err
		}
		result.LatestAvailablePatchVersion = nextTag.Name
		result.LatestGrade = nextTag.FreshnessGrade

	} else {
		// need to find the actual persistent tag for this floating tag
		for j, t := range tags {
			mr := regexp.MustCompile("^v?" + majorMinorVersion + "(\\W)+")
			if !mr.MatchString(t.Name) {
				log.Info("skipping tag ", t.Name, " as it does not match on major minor patch version "+majorMinorVersion+".*")
				continue
			}
			registryTagImage, err := i.imageGetter.Get(image.RegistryPath + ":" + t.Name)
			if err != nil {
				return result, errors.Wrap(err, "failed to get image details from registry for image "+image.RegistryPath+":"+t.Name)
			}

			if registryTagImage.Hash == clusterImageDigests.SHADigest {
				if t.Name == "latest" && len(tags) > 1 {
					// we continue as there will be an actual specific image version that matches
					continue
				}
				result.CurrentVersion = t.Name
				result.CurrentGrade = t.FreshnessGrade
				result.UpToDateWithFloatingTag = floatingTagImage.Hash == clusterImageDigests.SHADigest
				var (
					nextTag rhcc.Tag
					err     error
				)
				if j == 0 {
					// the current tag is the latest available tag
					nextTag = tags[0]
				} else {
					//check for latest patch version
					nextTag, err = findNextPatchImage(tags[:j+1], majorMinorVersion)
					if err != nil {
						return result, err
					}
				}
				result.LatestAvailablePatchVersion = nextTag.Name
				result.LatestGrade = nextTag.FreshnessGrade
				break
			}
		}

	}
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

func findNextPatchImage(tags []rhcc.Tag, majorMinorVersion string) (rhcc.Tag, error) {
	for i := range tags {
		match, err := regexp.MatchString("^v?"+majorMinorVersion+"(\\W)+", tags[i].Name)
		if err != nil {
			return rhcc.Tag{}, errors.Wrap(err, "failed to compile regex ^v?"+majorMinorVersion+"(\\W)+")
		}
		if match {
			// they are in order so break out now
			return tags[i], nil
		}
	}
	// if we don't find a newer one we are on the latest one take the last one in the array as this is the index where we found the match
	return tags[len(tags)-1], nil
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

func (i *ImageService) isPersistentTag(tags []rhcc.Tag, tag string) (bool, int) {
	for i, t := range tags {
		if t.Name == tag {
			return t.Type == "persistent", i
		}
	}
	return false, -1
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
