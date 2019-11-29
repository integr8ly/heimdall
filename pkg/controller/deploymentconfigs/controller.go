package deploymentconfigs

import (
	"github.com/integr8ly/heimdall/pkg/cluster"
	"github.com/integr8ly/heimdall/pkg/controller/validation"
	"github.com/integr8ly/heimdall/pkg/domain"
	"github.com/integr8ly/heimdall/pkg/registry"
	"github.com/integr8ly/heimdall/pkg/rhcc"
	v1 "github.com/openshift/api/apps/v1"
	apps "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	v12 "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	imagesv1 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	v13 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	"github.com/pkg/errors"
	v14 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
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
	registryImageService := registry.NewImagesService(&registry.Client{}, &rhcc.Client{}, &rhcc.Client{})

	return add(mgr, newReconciler(mgr, client, dcClient, isClient, registryImageService))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, k8sClient kubernetes.Interface, dcClient *v12.AppsV1Client, isClient *v13.ImageV1Client, riService *registry.ImageService) reconcile.Reconciler {
	clusterImageService := cluster.NewImageService(k8sClient, isClient)
	return &ReconcileDeploymentConfig{
		client:     mgr.GetClient(),
		scheme:     mgr.GetScheme(),
		dcClient:   dcClient,
		isClient:   isClient,
		podService: cluster.NewPods(mgr.GetClient()),
		reportService: &Reports{
			clusterImageService:  clusterImageService,
			registryImageService: riService,
			dcClient:             dcClient,
		},
		imageService: clusterImageService,
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
	client       client.Client
	scheme       *runtime.Scheme
	dcClient     v12.AppsV1Interface
	isClient     v13.ImageV1Interface
	podService   *cluster.Pods
	imageService *cluster.ImageService
	// turn into interfaces
	reportService *Reports
}

func (r *ReconcileDeploymentConfig) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// as we will watch all deployment configs we want to check if this is a deployment config we should care about.
	// we can label the deployment with last run time and we will see it again immediately but will then reque it based on the next check time
	dc, err := r.dcClient.DeploymentConfigs(request.Namespace).Get(request.Name, v14.GetOptions{})
	if err != nil {
		log.Error(err, "failed to get deployment config "+request.Namespace+"  "+request.Name)
		return reconcile.Result{}, err
	}
	if _, ok := dc.Labels[domain.HeimdallMonitored]; !ok {
		return reconcile.Result{}, nil
	}
	images, err := r.reportService.GetImages(dc)
	if err != nil {
		log.Error(err, "failed to get images for deployment config when checking if should run check again")
		return reconcile.Result{}, err
	}
	should, err := validation.ShouldCheck(dc, images)
	if err != nil {
		if validation.IsParseErr(err) {
			delete(dc.Annotations, domain.HeimdallLastChecked)
			if _, err := r.dcClient.DeploymentConfigs(request.Namespace).Update(dc); err != nil {
				// in this case we will requeue log the error and requeue to ensure we dont keep retrying the checks
				log.Error(err, " failed to label deployment config "+request.Namespace+" "+request.Name)
				return reconcile.Result{}, nil
			}
			return reconcile.Result{}, nil
		}
	}
	if !should {
		//return and we will see it again once it changes or 4 hrs passes
		log.Info("critera for re checking " + dc.Name + " not met")
		return reconcile.Result{RequeueAfter: requeAfterFourHours}, nil
	}

	log.Info("deployment config " + dc.Name + " in namespace " + dc.Namespace + " is being monitored by heimdall")
	// get the deployment config and work through the images we discover
	reports, err := r.reportService.Generate(request.Namespace, request.Name)
	if err != nil {
		log.Error(err, "failed to generate a report for images in dc "+request.Name+" in namespace "+request.Namespace)
		return reconcile.Result{RequeueAfter: requeAfterFourHours}, nil
	}
	// after the report has been run we want to annotate our dc with information. If we fail here we may end up re running the report.
	// reports can take some time so get a fresh dc copy
	dc, err = r.dcClient.DeploymentConfigs(request.Namespace).Get(request.Name, v14.GetOptions{})
	if err != nil {
		log.Error(err, "failed to get deployment config "+request.Namespace+"  "+request.Name)
		return reconcile.Result{}, nil
	}
	// have to use annotation as labels have strict length and format
	if dc.Annotations == nil {
		dc.Annotations = map[string]string{}
	}
	log.Info("generated reports for deployment ", "reports", len(reports), "namespace", request.Namespace, "name", request.Name)
	checked := []string{}
	for _, rep := range reports {
		checked = append(checked, rep.ClusterImage.SHA256Path)
		if err := r.podService.LabelPods(&rep); err != nil {
			log.Error(err, "failed to label pod ")
			return reconcile.Result{}, nil
		}
	}
	dc.Annotations[domain.HeimdallLastChecked] = time.Now().Format(domain.TimeFormat)
	dc.Annotations[domain.HeimdallImagesChecked] = strings.Join(checked, ",")
	if _, err := r.dcClient.DeploymentConfigs(request.Namespace).Update(dc); err != nil {
		// in this case we will requeue log the error and requeue to ensure we dont keep retrying the checks
		log.Error(err, " failed to label deployment config "+request.Namespace+" "+request.Name)
		return reconcile.Result{}, nil
	}
	// ensure we see this dc 4 hours from now or when it next changes
	return reconcile.Result{RequeueAfter: requeAfterFourHours}, nil
}
