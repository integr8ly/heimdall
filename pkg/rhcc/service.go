package rhcc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

const host = "https://rhcc-api.redhat.com/rest/v1/repository/%s/%s"

func ImageMeta(registry, org, imageName string) (*ContainerRepository, error) {
	cr := &ContainerRepository{}
	// seems to need double encoding
	image := url.QueryEscape(url.QueryEscape(org + "/" + imageName))
	url := fmt.Sprintf(host, registry, image)
	url += "/images"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected response from rhcc api " + resp.Status)
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(cr); err != nil {
		return nil, err
	}
	//for _,c := range cr.Processed{
	//
	//}
	fmt.Println(url, cr)
	return &ContainerRepository{}, nil
}
