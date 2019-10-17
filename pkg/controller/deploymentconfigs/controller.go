package deploymentconfigs

import (
	"github.com/integr8ly/heimdall/pkg/cluster"
	"github.com/integr8ly/heimdall/pkg/registry"
	"github.com/integr8ly/heimdall/pkg/rhcc"
	v1 "github.com/openshift/api/apps/v1"
	apps "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	v12 "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	imagesv1 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	v13 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

var log = logf.Log.WithName("controller_deploymentconfigs")

const requeAfterFourHours = time.Hour * 4


// Add creates a new ImageMonitor Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	if err := v1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}
	dcClient, err := apps.NewForConfig(mgr.GetConfig())
	if err != nil {
		return errors.Wrap(err, "failed to create dc client")
	}

	isClient, err := imagesv1.NewForConfig(mgr.GetConfig())
	if err != nil {
		return errors.Wrap(err, "failed to create images client")
	}
	client, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client")
	}
	registryImageService := registry.NewImagesService(&registry.Client{}, &rhcc.Client{},&rhcc.Client{})

	return add(mgr, newReconciler(mgr, client, dcClient, isClient, registryImageService))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, k8sClient kubernetes.Interface, dcClient *v12.AppsV1Client, isClient *v13.ImageV1Client, riService *registry.ImageService) reconcile.Reconciler {
clusterImageService := cluster.NewImageService(k8sClient, isClient)
	return &ReconcileDeploymentConfig{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		dcClient:            dcClient,
		isClient:            isClient,
		podService:          cluster.NewPods(mgr.GetClient()),
		reportService: &Reports{
			clusterImageService:  clusterImageService,
			registryImageService: riService,
			dcClient:dcClient,
		},
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {

	c, err := controller.New("deploymentconfig-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}
	return c.Watch(&source.Kind{Type: &v1.DeploymentConfig{}}, &handler.EnqueueRequestForObject{})
}

var _ reconcile.Reconciler = &ReconcileDeploymentConfig{}

// ReconcileImageMonitor reconciles a ImageMonitor object
type ReconcileDeploymentConfig struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client              client.Client
	scheme              *runtime.Scheme
	dcClient            v12.AppsV1Interface
	isClient            v13.ImageV1Interface
	podService          *cluster.Pods
	// turn into interfaces
	reportService *Reports
}




func (r *ReconcileDeploymentConfig) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// get the deployment config and work through the images we discover
	report, err :=r.reportService.Generate(request.Namespace, request.Name)
	if err != nil{
		log.Error(err,"failed to generate a report for images in dc " + request.Name + " in namespace " + request.Namespace)
		return reconcile.Result{RequeueAfter: requeAfterFourHours}, nil
	}
	for _,rep := range report{
		if err := r.podService.LabelPods(&rep); err != nil{
			log.Error(err,"failed to label pod ")
			return reconcile.Result{RequeueAfter: requeAfterFourHours}, nil
		}
	}


	return reconcile.Result{RequeueAfter:requeAfterFourHours}, nil
}
