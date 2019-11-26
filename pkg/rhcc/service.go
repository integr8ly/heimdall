package rhcc

import (
	"encoding/json"
	"github.com/pkg/errors"

	"fmt"
	"github.com/integr8ly/heimdall/pkg/domain"
	"net/http"
	"net/url"
	"sort"
	"time"
)

const host = "https://rhcc-api.redhat.com/rest/v1"
const images = "%s/repository/%s/%s/images"
const image = "%s/repository/%s/%s/images/%s?architecture="

type Client struct {
}

type Tag struct {
	Name           string
	Added          string
	TimeAdded      int64
	FreshnessGrade string
	// can be floating or persistent
	Type string
}

func (c *Client) AvailableTagsSortedByDate(org string) ([]Tag, error) {
	cr := &ContainerRepository{}
	// seems to need double encoding
	image := url.QueryEscape(url.QueryEscape(org))
	// done to allow us to call the API without the need for credentials (should revisit)
	url := fmt.Sprintf(images, host, "registry.access.redhat.com", image)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected response from rhcc api " + resp.Status)
	}
	var format = "20060102T15:04:05.000-0700"
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(cr); err != nil {
		return nil, err
	}
	// sort by added date
	//20191125T09:53:00.000-0500
	var tags []Tag
	var freshnessGrade *FreshnessGrade
	for _, i := range cr.Processed[0].Images {
		for _, fg := range i.FreshnessGrades {
			sd, _ := time.Parse(format, fg.StartDate)
			if sd.Before(time.Now()) {
				ed, _ := time.Parse(format, fg.EndDate)
				if !ed.After(time.Now()) {
					freshnessGrade = fg
					break
				}
			}

		}
		// add freshness grade based on whether end date has passed or not
		for _, r := range i.Repositories {
			for _, t := range r.Tags {
				var tagType string
				if len(t.TagHistory) == 1 {
					tagType = t.TagHistory[0].TagType
				}
				tag := Tag{
					Name:  t.Name,
					Added: t.AddedDate,
					Type:  tagType,
				}
				addedTime, err := time.Parse(format, t.AddedDate)
				if err != nil {
					return nil, errors.Wrap(err, "failed to parse time image was pushed")
				}
				if freshnessGrade != nil {
					tag.FreshnessGrade = freshnessGrade.Grade
				}
				tag.TimeAdded = addedTime.Unix()
				tags = append(tags, tag)
			}
		}
	}

	sort.Slice(tags, func(i, j int) bool {
		return tags[i].TimeAdded > tags[j].TimeAdded
	})
	return tags, nil
}

func (c *Client) CVES(org, tag string) ([]domain.CVE, error) {
	if org == "" || tag == "" {
		return nil, errors.New("expected and org and a tag but got org  " + org + " tag " + tag)
	}
	cri := &ContainerRepositoryImage{}
	i := url.QueryEscape(url.QueryEscape(org))
	url := fmt.Sprintf(image, host, "registry.access.redhat.com", i, tag)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected response from rhcc api " + resp.Status)
	}

	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(cri); err != nil {
		return nil, err
	}
	var cves []domain.CVE
	// should only be one image as we used specific tag
	for _, v := range cri.Processed[0].Images[0].VulnerabilitiesRef {
		cves = append(cves, domain.CVE{AdvisoryID: v.AdvisoryID, Severity: v.Severity, ID: v.CveID})
	}
	return cves, nil
}
