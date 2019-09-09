package registry

//https://rhcc-api.redhat.com/rest/v1/repository/registry.access.redhat.com/amq7%252Famq-online-1-auth-plugin/images

import (
	"fmt"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type Client struct {
	Host string
	Auth string
}

func (c *Client) Image(image string) (*Image, error) {
	//remote.WithAuth()
	i, r, err := GetImage(image)
	if err != nil {
		return nil, err
	}
	fmt.Println("image", i, "ref", r)
	return &Image{}, nil
}

func GetImage(r string) (v1.Image, name.Reference, error) {
	ref, err := name.ParseReference(r)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing reference %q: %v", r, err)
	}
	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return nil, nil, fmt.Errorf("reading image %q: %v", ref, err)
	}
	//fmt.Printf("img %v  ref %v", img.Digest(),ref)
	return img, ref, nil
}
