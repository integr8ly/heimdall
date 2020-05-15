package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/integr8ly/heimdall/pkg/cluster"
	"github.com/integr8ly/heimdall/pkg/controller/deploymentconfigs"
	"github.com/integr8ly/heimdall/pkg/controller/deployments"
	"github.com/integr8ly/heimdall/pkg/controller/statefulset"
	"github.com/integr8ly/heimdall/pkg/domain"
	"github.com/integr8ly/heimdall/pkg/registry"
	"github.com/integr8ly/heimdall/pkg/rhcc"
	"github.com/jedib0t/go-pretty/table"
	v1 "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	imagesv1 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func main() {
	namespacePtr := flag.String("namespaces", "", "the namespaces to check")
	namespacePatternPtr := flag.String("namespace-pattern", "", "a go compilant regular expression to include only matching namespaces")
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
	clusterIS := cluster.NewImageService(client, isClient)
	registryIS := registry.NewImagesService(&registry.Client{}, &rhcc.Client{}, &rhcc.Client{})
	dcReport := deploymentconfigs.NewReport(clusterIS, registryIS, dcClient)
	deploymentReport := deployments.NewReport(clusterIS, registryIS, client.AppsV1())
	statefulSetReport := statefulset.NewReport(clusterIS, registryIS, client.AppsV1())
	var reports []domain.ReportResult
	namespaces, err := getNamespaces(client, namespacePtr)
	if err != nil {
		log.Fatalf("error getting namespaces: %v", err)
	}
	namespaces, err = filterNamespaces(namespaces, namespacePatternPtr)
	if err != nil {
		log.Fatalf("error filtering namespaces with pattern %s: %v", *namespacePatternPtr, err)
	}
	for _, n := range namespaces {
		nsReports, err := accumulateReports(n, *componentPtr,
			dcReport.Generate,
			deploymentReport.Generate,
			statefulSetReport.Generate,
		)
		if err != nil {
			log.Println("failed to generate image report " + err.Error())
		}
		reports = append(reports, nsReports...)
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)

		t.AppendHeader(table.Row{"component", "Image", "Image Hash", "Image Stream", "Tag", "UpTo Date With Tag", "Persistent Image Tag", "Latest Patch Tag", "Floating Tag", "Using Floating Tag", "Upto Date with Floating Tag", "Critical CVEs", "Important CVEs", "Moderate CVEs"})
		for i := range reports {
			t.AppendRows([]table.Row{
				{reports[i].Component,
					reports[i].ClusterImage.OrgImagePath,
					reports[i].ClusterImage.GetSHAFromPath(),
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

// getNamespaces obtains a slice of namespaces to inspect based on the presence
// and contents of the `namespaceFlag` passed as a command argument
func getNamespaces(client *kubernetes.Clientset, namespaceFlag *string) ([]string, error) {
	if namespaceFlag != nil && *namespaceFlag != "" {
		return strings.Split(*namespaceFlag, ","), nil
	}

	namespaceList, err := client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error querying namespaces: %w", err)
	}

	result := make([]string, len(namespaceList.Items))
	for i, ns := range namespaceList.Items {
		result[i] = ns.Name
	}

	return result, nil
}

// filterNamespaces returns a slice of strings from the `namespaces` parameter
// that match the `namespacePatternFlag` regex. If `namespacePatternFlag` has
// no value, all namespaces are included.
func filterNamespaces(namespaces []string, namespacePatternFlag *string) ([]string, error) {
	if namespacePatternFlag == nil || *namespacePatternFlag == "" {
		return namespaces, nil
	}

	if len(namespaces) == 0 {
		return namespaces, nil
	}

	pattern := strings.Trim(*namespacePatternFlag, "\"")

	result := make([]string, 0, len(namespaces))
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	for _, namespace := range namespaces {
		if re.MatchString(namespace) {
			result = append(result, namespace)
		}
	}

	return result, nil
}

// accumulateReports takes a variadic list of functions that generate reports
// and invokes them passing the same given ns and name, and accumulates all
// the results in a single slice. If any of the generate function fails, the
// function returns the error.
func accumulateReports(ns, name string, generateFns ...func(string, string) ([]domain.ReportResult, error)) ([]domain.ReportResult, error) {
	result := []domain.ReportResult{}

	for _, generateFn := range generateFns {
		reports, err := generateFn(ns, name)
		if err != nil {
			return result, err
		}

		result = append(result, reports...)
	}

	return result, nil
}
