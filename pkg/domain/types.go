package domain

import (
	v1 "github.com/openshift/api/image/v1"
	"regexp"
	"strings"
	"time"
)

type ClusterImage struct {
	// registry.redhat.io/3scale-amp26/system:latest
	//
	Component       string
	FullPath        string
	OrgImagePath    string
	Tag             string
	ImageName       string
	RegistryPath    string
	Org             string
	SHA256Path      string
	Pods            []PodAndContainerRef
	FromImageStream bool
	ImageStreamTag  *v1.ImageStreamTag
}

func (ci *ClusterImage) GetSHAFromPath() string {
	if ci.SHA256Path == "" {
		return ""
	}
	parts := strings.Split(ci.SHA256Path, ":")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

type PodAndContainerRef struct {
	Name       string
	Namespace  string
	Containers []string
}

func (i *ClusterImage) IsSHATag() bool {
	reg := regexp.MustCompile("@?sha256")
	return reg.MatchString(i.FullPath)
}

func (i *ClusterImage) String() string {
	return i.FullPath + " : " + i.OrgImagePath + " : " + i.Tag + ":" + i.Org + " : " + i.RegistryPath + " : " + i.Org
}

func NewRemoteImageDigest(digest, algo string) *RemoteImageDigest {
	return &RemoteImageDigest{
		Hash:      digest,
		Algorithm: algo,
	}
}

type RemoteImageDigest struct {
	Algorithm string
	Hash      string
}

type CVE struct {
	Severity   string
	ID         string
	AdvisoryID string
}

type ReportResult struct {
	Component                   string
	ActualImageRef              string
	ResolvableCVEs              []CVE
	CurrentVersion              string
	LatestAvailablePatchVersion string
	FloatingTag                 string
	UsingFloatingTag            bool
	CurrentGrade                string
	LatestGrade                 string
	UpToDateWithOwnTag          bool
	UpToDateWithFloatingTag     bool
	ClusterImage                *ClusterImage
}

func (cr ReportResult) GetResolvableCriticalCVEs() []CVE {
	crit := []CVE{}
	for _, c := range cr.ResolvableCVEs {
		if strings.ToLower(c.Severity) == "critical" {
			crit = append(crit, c)
		}
	}
	return crit
}
func (cr ReportResult) GetResolvableImportantCVEs() []CVE {
	important := []CVE{}
	for _, c := range cr.ResolvableCVEs {
		if strings.ToLower(c.Severity) == "important" {
			important = append(important, c)
		}
	}
	return important
}

func (cr ReportResult) GetResolvableModerateCVEs() []CVE {
	important := []CVE{}
	for _, c := range cr.ResolvableCVEs {
		if strings.ToLower(c.Severity) == "moderate" {
			important = append(important, c)
		}
	}
	return important
}

func (cr ReportResult) String() string {
	return "currentVersion :" + cr.CurrentVersion + " LatestAvailablePatchVersion: " + cr.LatestAvailablePatchVersion + " Floating Tag: " + cr.FloatingTag + " actual image ref " + cr.ActualImageRef
}

const (
	HeimdallMonitored      = "heimdall.monitored"
	HeimdallLastChecked    = "heimdall.lastcheck"
	HeimdallImagesChecked  = "heimdall.imageschecked"
	TimeFormat             = time.RFC822Z
	MinRecheckIntervalMins = 30
)
