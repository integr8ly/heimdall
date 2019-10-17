package main

import (
	"flag"
	"github.com/integr8ly/heimdall/pkg/cluster"
	"github.com/integr8ly/heimdall/pkg/controller/deploymentconfigs"
	"github.com/integr8ly/heimdall/pkg/controller/deployments"
	"github.com/integr8ly/heimdall/pkg/domain"
	"github.com/integr8ly/heimdall/pkg/registry"
	"github.com/integr8ly/heimdall/pkg/rhcc"
	"github.com/jedib0t/go-pretty/table"
	"github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	imagesv1 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"os"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"strings"
)

//curl 'https://rhcc-api.redhat.com/rest/v1/repository/registry.access.redhat.com/amq7%252Famq-online-1-api-server/images' -H 'accept: application/json' -H 'Origin: https://access.redhat.com' -H 'Sec-Fetch-Mode: cors' --compressed

func main() {
	namespacePtr := flag.String("namespaces", "", "the namespaces to check")
	componentPtr := flag.String("component", "*", "the dc or deployment name to check in the namespace")
	labelPodsPtr := flag.String("label-pods", "false", "add labels to the pods with the info discovered")
	flag.Parse()

	conf := config.GetConfigOrDie()
	client, err := kubernetes.NewForConfig(conf)
	if err != nil {
		log.Fatal("failed to get a client", err)
	}

	dcClient, err := v1.NewForConfig(conf)
	if err != nil {
		log.Fatal("failed to create deploymentconfig client")
	}

	isClient, err := imagesv1.NewForConfig(conf)
	if err != nil {
		log.Fatal("failed to create image stream client")
	}
	clusterIS := cluster.NewImageService(client,isClient)
	registryIS := registry.NewImagesService(&registry.Client{},&rhcc.Client{},&rhcc.Client{})
	dcReport := deploymentconfigs.NewReport(clusterIS,registryIS, dcClient)
	deploymentReport := deployments.NewReport(clusterIS,registryIS, client.AppsV1())
	var reports []domain.ReportResult
	namespaces := strings.Split(*namespacePtr, ",")
	for _, n := range namespaces {

		dcReports, err := dcReport.Generate(n, *componentPtr)
		if err != nil {
			log.Println("failed to generate image report " + err.Error())
		}
		reports = append(reports, dcReports...)
		deploymentReports, err := deploymentReport.Generate(n, *componentPtr)
		reports = append(reports, deploymentReports...)
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)

		t.AppendHeader(table.Row{"component", "Image", "Image Stream", "Tag", "UpTo Date With Tag", "Persistent Image Tag", "Latest Patch Tag", "Floating Tag", "Using Floating Tag", "Upto Date with Floating Tag", "Critical CVEs", "Important CVEs", "Moderate CVEs"})
		for i := range reports {
			t.AppendRows([]table.Row{
				{reports[i].Component,
					reports[i].ClusterImage.OrgImagePath,
					reports[i].ClusterImage.FromImageStream,
					reports[i].ClusterImage.Tag,
					reports[i].UpToDateWithOwnTag,
					reports[i].CurrentVersion,
					reports[i].LatestAvailablePatchVersion,
					reports[i].FloatingTag,
					reports[i].UsingFloatingTag,
					reports[i].UpToDateWithFloatingTag,
					len(reports[i].GetResolvableCriticalCVEs()),
					len(reports[i].GetResolvableImportantCVEs()),
					len(reports[i].GetResolvableModerateCVEs())},
			})
		}

		if *labelPodsPtr == "true" {
			c, err := client2.New(conf, client2.Options{})
			if err != nil {
				log.Println("failed to get a client for labelling pods ", err.Error())
				return
			}
			podService := cluster.NewPods(c)
			for _, r := range reports {
				if err := podService.LabelPods(&r); err != nil {
					log.Println("failed to label pods ", err)
				}
			}
		}
		t.Render()
		//time.Sleep(time.Minute * 3)
	}
}
