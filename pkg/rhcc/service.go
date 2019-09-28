package rhcc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

//https://rhcc-api.redhat.com/rest/v1/repository/registry.access.redhat.com/amq7%252Famq-online-1-api-server/images
//https://rhcc-api.redhat.com/rest/v1/repository/registry.access.redhat.com/redhat-sso-7%252Fsso73-openshift/images/1.0-13?architecture=

const host = "https://rhcc-api.redhat.com/rest/v1"
const images = "%s/repository/%s/%s/images"
const image = "%s/repository/%s/%s/images/%s?architecture="

func AvailableTags( org string) ([]string, error) {
	cr := &ContainerRepository{}
	// seems to need double encoding
	image := url.QueryEscape(url.QueryEscape(org))
	url := fmt.Sprintf(images,host, "registry.access.redhat.com", image)
	fmt.Println("calling images url ", url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected response from rhcc api " + resp.Status)
	}

	defer resp.Body.Close()
	//data, err :=ioutil.ReadAll(resp.Body)
	//if err != nil{
	//	return nil, err
	//}
	//fmt.Println(string(data))
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(cr); err != nil {
		return nil, err
	}
	//for _,c := range cr.Processed{
	//
	//}
	//fmt.Println(url, cr.Processed[0].Images)
	var tags []string
	for _ , i := range cr.Processed[0].Images{
		for _, r := range i.Repositories{
			for _, t := range r.Tags{
				tags = append(tags, t.Name)
			}
		}
	}
	return tags, nil
}

func CVES(org, tag string )([]CVE, error)  {
	cri := &ContainerRepositoryImage{}
	i := url.QueryEscape(url.QueryEscape(org))
	url := fmt.Sprintf(image,host, "registry.access.redhat.com", i, tag)
	fmt.Println("calling image url ", url)
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
	var cves []CVE
	// should only be one image as we used specific tag
	for _, v := range cri.Processed[0].Images[0].VulnerabilitiesRef{
		cves = append(cves, CVE{AdvisoryID:v.AdvisoryID, Severity:v.Severity, CveID:v.CveID})
	}
	return cves, nil
}
