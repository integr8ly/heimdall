package e2e

import (
	goctx "context"
	"fmt"
	"github.com/integr8ly/heimdall/pkg/apis/imagemonitor/v1alpha1"
	"golang.org/x/net/context"
	"k8s.io/api/apps/v1beta1"
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
	"time"

	"github.com/integr8ly/heimdall/pkg/apis"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	errorUtil "github.com/pkg/errors"
)

const (
	imagemonitorName = "test-imagemonitor"
	cvePodName       = "test-cve-pod"
)

var (
	retryInterval = time.Second * 20
	timeout       = time.Minute * 5
)

func TestHeimdall(t *testing.T) {

	imagemonitorList := &v1alpha1.ImageMonitor{}
	if err := framework.AddToFrameworkScheme(apis.AddToScheme, imagemonitorList); err != nil {
		t.Fatalf("failed to add Imagemonitor custom resource scheme to framework: %v", err)
	}

	t.Run("heimdall-operator-test", func(t *testing.T) {
		t.Run("Cluster", OperatorTest)
	})
}

func OperatorTest(t *testing.T) {
	t.Parallel()
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	err := ctx.InitializeClusterResources(getCleanupOptions(t))
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}
	t.Log("initialized cluster resources")

	f := framework.Global

	monitor, namespace, err := getImagemonitor(framework.TestCtx{})
	if err != nil {
		t.Fatalf("failed to get imagemonitor: %v", err)
	}

	if err := imagemonitorCreate(t, f, monitor, namespace); err != nil {
		t.Fatalf("failed to create imagemonitor resource: %v", err)
	}

	if err := deployKnownCVEPods(t, framework.TestCtx{}, f); err != nil {
		t.Fatalf("failed to create known cve pod deployment: %v", err)
	}

	if err := checkPodsGetLabelled(t, framework.TestCtx{}, f); err != nil {
		t.Fatalf("failed to label known cve pod deployment: %v", err)
	}

}

func checkPodsGetLabelled(t *testing.T, ctx framework.TestCtx, f *framework.Framework) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return errorUtil.Wrapf(err, "could not get namespace")
	}

	labels := make(map[string]string)
	labels["app"] = "cvepod"
	podList := &v12.PodList{}
	labelSelector := k8slabels.SelectorFromSet(labels)
	listOps := &client.ListOptions{Namespace: namespace, LabelSelector: labelSelector}
	err = f.Client.List(context.TODO(), podList, listOps)
	if err != nil {
		return errorUtil.Wrapf(err, "Failed to list pods")
	}

	cvePod := &v12.Pod{}

	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		if err = f.Client.Get(goctx.TODO(), types.NamespacedName{Namespace: namespace, Name: podList.Items[0].Name}, cvePod); err != nil {
			fmt.Print(cvePod.Name)

			if val, ok := cvePod.Labels["heimdall.updatedImageAvailable"]; ok {
				t.Logf("found label: %v", val)
				return true, nil
			}
		}
		return true, errorUtil.Wrapf(err, "could not find label")
	})
	return nil
}

func getCVEContainer() v12.Container {
	return v12.Container{
		Name:  cvePodName,
		Image: "registry.redhat.io/rhscl/httpd-24-rhel7:2.4-73",
	}
}

func getImagemonitor(ctx framework.TestCtx) (*v1alpha1.ImageMonitor, string, error) {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return nil, "", errorUtil.Wrapf(err, "could not get namespace")
	}

	return &v1alpha1.ImageMonitor{
		ObjectMeta: v1.ObjectMeta{
			Name:      imagemonitorName,
			Namespace: namespace,
		},
		Spec: v1alpha1.ImageMonitorSpec{
			ExcludePattern: "",
		},
	}, namespace, nil
}

func deployKnownCVEPods(t *testing.T, ctx framework.TestCtx, f *framework.Framework) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return errorUtil.Wrapf(err, "could not get namespace")
	}

	cvePodContainer := getCVEContainer()

	labels := make(map[string]string)
	labels["app"] = "cvepod"
	labels["heimdall.monitored"] = "true"

	var cont []v12.Container
	containers := append(cont, cvePodContainer)

	lselector := &v1.LabelSelector{
		MatchLabels: labels,
	}

	cveDeployment := &v1beta1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      cvePodName,
			Namespace: namespace,
		},
		Spec: v1beta1.DeploymentSpec{
			Selector: lselector,
			Template: v12.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: labels,
				},
				Spec: v12.PodSpec{
					Containers: containers,
				},
			},
		},
		Status: v1beta1.DeploymentStatus{},
	}

	if err := f.Client.Create(goctx.TODO(), cveDeployment, getCleanupOptions(t)); err != nil {
		return errorUtil.Wrapf(err, "could not create cve deployment")
	}

	pcr := &v12.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      cvePodName,
			Namespace: namespace,
		},
		Spec: v12.PodSpec{},
	}
	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Namespace: namespace, Name: cvePodName}, pcr); err != nil {
			return true, errorUtil.Wrapf(err, "could not get cve deployment")
		}
		return true, nil
	})
	return nil
}

func imagemonitorCreate(t *testing.T, f *framework.Framework, testImagemonitor *v1alpha1.ImageMonitor, namespace string) error {
	if err := f.Client.Create(goctx.TODO(), testImagemonitor, getCleanupOptions(t)); err != nil {
		return errorUtil.Wrapf(err, "could not create example imagemonitor")
	}
	t.Logf("created %s resource", testImagemonitor.Name)

	// poll cr for complete status phase
	pcr := &v1alpha1.ImageMonitor{}
	_ = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Namespace: namespace, Name: imagemonitorName}, pcr); err != nil {
			return true, errorUtil.Wrapf(err, "could not get postgres cr")
		}
		return true, nil
	})
	return nil
}

func getCleanupOptions(t *testing.T) *framework.CleanupOptions {
	return &framework.CleanupOptions{
		TestContext:   framework.NewTestCtx(t),
		Timeout:       timeout,
		RetryInterval: retryInterval,
	}
}
