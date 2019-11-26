package registry

import (
	"fmt"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/integr8ly/heimdall/pkg/domain"
	"github.com/pkg/errors"
	"os"
)

type Client struct {
	Host string
	Auth string
}

func (c *Client) Get(r string) (*domain.RemoteImageDigest, error) {
	remote.WithAuth(c)
	ref, err := name.ParseReference(r)
	if err != nil {
		return nil, fmt.Errorf("parsing reference %q: %v", r, err)
	}
	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return nil, fmt.Errorf("reading image %q: %v", ref, err)
	}
	digest, err := img.Digest()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get image digest")
	}
	return domain.NewRemoteImageDigest(digest.Hex, digest.Algorithm), nil
}
func (c *Client) Authorization() (string, error) {
	return os.Getenv("REGISTRY_TOKEN"), nil
}
