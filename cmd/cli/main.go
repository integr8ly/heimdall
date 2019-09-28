package main

import (
	"fmt"
	"github.com/integr8ly/heimdall/pkg/cluster"
	"github.com/integr8ly/heimdall/pkg/registry"
	"github.com/integr8ly/heimdall/pkg/rhcc"
	"github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	imagesv1 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"log"
	"os"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"strings"
)

//curl 'https://rhcc-api.redhat.com/rest/v1/repository/registry.access.redhat.com/amq7%252Famq-online-1-api-server/images' -H 'accept: application/json' -H 'Origin: https://access.redhat.com' -H 'Sec-Fetch-Mode: cors' --compressed

func main() {
	conf := config.GetConfigOrDie()
	client, err := kubernetes.NewForConfig(conf)
	if err != nil {
		log.Fatal("failed to get a client", err)
	}

	if len(os.Args) < 2 {
		log.Fatal("unknown command \n check")
	}
	dcClient, err := v1.NewForConfig(conf)
	if err != nil {
		log.Fatal("failed to create deploymentconfig client")
	}

	isClient, err := imagesv1.NewForConfig(conf)
	if err != nil {
		log.Fatal("failed to create image stream client")
	}

	clusterService := cluster.NewImageService(client, dcClient, isClient)
	switch os.Args[1] {
	case "check":
		if err := check(clusterService, os.Args[2:]); err != nil {
			log.Fatal("command failed ", err)
		}
		return
	default:
		log.Fatal("unknown command ", os.Args[1])

	}
}

func check(ds *cluster.ImageService, args []string) error {
	// for each image
	// check the image name and tag manifest and store the sha
	// check the imageID manifest and get the sha
	// the two shas should match
	if len(args) < 1 {
		return errors.New("missing arguments. Expected a ns and pattern to match")
	}
	if len(args) ==1 {
		args = append(args, ".*")
	}

	ns := args[0]
	fmt.Println("regex" , args[1])
	pattern := regexp.MustCompile(args[1])
	images, err := ds.FindImages(ns, pattern)
	if err != nil {
		return err
	}
	for _, i := range images {
		if strings.Contains(i.RegistryPath, "quay"){
			fmt.Println("ignoring quay registry")
			continue
		}
		fmt.Println("" +
			"---- CHECK RESULT ---- " +
			"")
		fmt.Println("Checking image " + i.RegistryPath + ":" + i.Tag)
		img, _, err := registry.GetImage(i.RegistryPath + ":" + i.Tag)
		if err != nil {
			return errors.Wrap(err, "failed to get image from registry "+i.RegistryPath+":"+i.Tag)
		}
		_, err = img.RawManifest()
		if err != nil {
			return err
		}

		//fmt.Println("**manifest**")
		//fmt.Println(string(rm))
		//fmt.Println("**manifest**")

		latestHash, err := img.Digest()
		if err != nil {
			return errors.Wrap(err, "failed to get digest of latest image")
		}
		img, _, err = registry.GetImage(i.RegistryPath + ":" + i.SHA)
		if err != nil {
			return errors.Wrap(err, "failed to get image from registry "+i.RegistryPath+":"+i.SHA)
		}
		_, err = img.RawManifest()
		if err != nil {
			return err
		}

		//fmt.Println("*running manifest**")
		//fmt.Println(string(rm))
		//fmt.Println("**running manifest**")
		runningImgHash, err := img.Digest()
		if err != nil {
			return err
		}
		fmt.Println("registry ", i.Registry, "org ", i.OrgPath, " image ", i.ImageName)
		// TODO can we go through each image in the registry to find which one this hash does match
		fmt.Println("Running hash: ", runningImgHash)
		fmt.Println("Latest Registry  hash: ", latestHash, "err", err)
		if latestHash == runningImgHash {
			fmt.Println("** RUNNING IMAGE IS UPTO DATE **")
		} else {
			fmt.Println("!XX RUNNING IMAGE IS NOT!! UPTO DATE XX!")

		}
		fmt.Println("-- END RESULT ---" +
			"")

		tags , err := rhcc.AvailableTags(i.OrgPath)
		if err != nil{
			return errors.Wrap(err, "failed to get tags")
		}
		fmt.Println("found tags in registry ", tags)
		for _, t := range tags{
			it, _, err := registry.GetImage(i.RegistryPath + ":" + t)
			if err != nil {
				return errors.Wrap(err, "failed to get image from registry "+i.RegistryPath+":"+i.Tag)
			}
			hash, err := it.Digest()
			if hash == runningImgHash{
				fmt.Println("actual image tag running in cluster is ", t)
				fmt.Println("checking CVEs")
				cves, err := rhcc.CVES(i.OrgPath, t)
				if err != nil{
					return errors.Wrap(err, "failed to get cves")
				}
				fmt.Println("found cves ", cves, len(cves))
				break
			}
		}
	}

	return nil

}
