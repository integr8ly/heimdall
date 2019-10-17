package deployments

import (
	"fmt"
	"github.com/integr8ly/heimdall/pkg/cluster"
	"github.com/integr8ly/heimdall/pkg/registry"
	"github.com/integr8ly/heimdall/pkg/rhcc"
	"github.com/pkg/errors"
	v12 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"time"
)

var log = logf.Log.WithName("controller_deployments")

const labelFormat = "heimdall.%s"

const requeAfterFourHours = time.Hour * 4

// Add creates a new ImageMonitor Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	client, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client")
	}
	registryImageService := registry.NewImagesService(&registry.Client{}, &rhcc.Client{},&rhcc.Client{})

	return add(mgr, newReconciler(mgr, client, registryImageService))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, k8sClient kubernetes.Interface,  riService *registry.ImageService) reconcile.Reconciler {
	clusterImageService := cluster.NewImageService(k8sClient, nil)
	return &ReconcileDeployment{
		client: mgr.GetClient(), scheme: mgr.GetScheme(),
		reportService: &Reports{
			clusterImageService:  clusterImageService,
			registryImageService: riService,
		},
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {

	c, err := controller.New("deployment-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}
	return c.Watch(&source.Kind{Type: &v12.Deployment{}}, &handler.EnqueueRequestForObject{})
}

func (r *ReconcileDeployment) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	fmt.Print("reconcile called for deployment ", request.Namespace, request.Name)
	return reconcile.Result{}, nil
}

var _ reconcile.Reconciler = &ReconcileDeployment{}

// ReconcileImageMonitor reconciles a ImageMonitor object
type ReconcileDeployment struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client              client.Client
	scheme              *runtime.Scheme
	// turn into interfaces
	reportService *Reports
}

//docker-registry.default.svc:5000/openshift/postgresql
//registry.redhat.io/rhscl/postgresql-96-rhel7


