package deployments

import (
	"context"
	"github.com/integr8ly/heimdall/pkg/cluster"
	"github.com/integr8ly/heimdall/pkg/controller/validation"
	"github.com/integr8ly/heimdall/pkg/domain"
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
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
	"time"
)

var log = logf.Log.WithName("controller_deployments")

const labelFormat = "heimdall.%s"

const requeAfterFourHours = time.Hour * 4

func Add(mgr manager.Manager) error {
	client, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client")
	}
	registryImageService := registry.NewImagesService(&registry.Client{}, &rhcc.Client{}, &rhcc.Client{})

	return add(mgr, newReconciler(mgr, client, registryImageService))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, k8sClient kubernetes.Interface, riService *registry.ImageService) reconcile.Reconciler {
	clusterImageService := cluster.NewImageService(k8sClient, nil)
	return &ReconcileDeployment{
		client: mgr.GetClient(), scheme: mgr.GetScheme(),
		reportService: &Reports{
			clusterImageService:  clusterImageService,
			registryImageService: riService,
			deploymentClient:     k8sClient.AppsV1(),
		},
		podService:   cluster.NewPods(mgr.GetClient()),
		imageService: clusterImageService,
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
	ctx := context.TODO()
	d := &v12.Deployment{}
	err := r.client.Get(context.TODO(), client.ObjectKey{Namespace: request.Namespace, Name: request.Name}, d)
	if err != nil {
		log.Error(err, "failed to get deployment in namespace "+request.Namespace+" with name  "+d.Name)
		return reconcile.Result{}, err
	}
	// ignore if not labeled
	if _, ok := d.Labels[domain.HeimdallMonitored]; !ok {
		return reconcile.Result{}, nil
	}
	images, err := r.reportService.GetImages(d)
	if err != nil {
		return reconcile.Result{}, err
	}
	should, err := validation.ShouldCheck(d, images)
	if err != nil {
		if validation.IsParseErr(err) {
			delete(d.Annotations, domain.HeimdallLastChecked)
			if err := r.client.Update(context.TODO(), d); err != nil {
				// in this case we will requeue log the error and requeue to ensure we dont keep retrying the checks
				log.Error(err, " failed to label deployment "+request.Namespace+" "+request.Name)
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, err
		}
	}
	if !should {
		log.Info("critera for re checking " + d.Name + " not met")
		return reconcile.Result{}, nil
	}

	log.Info("deployment " + d.Name + " in namespace " + d.Namespace + " is being monitored by heimdall")

	report, err := r.reportService.Generate(request.Namespace, request.Name)
	if err != nil {
		log.Error(err, "failed to generate a report for images in dc "+request.Name+" in namespace "+request.Namespace)
		return reconcile.Result{RequeueAfter: requeAfterFourHours}, nil
	}
	log.Info("generated reports for deployment ", "reports", len(report), "namespace", request.Namespace, "name", request.Name)
	// make sure we are upto date
	err = r.client.Get(context.TODO(), client.ObjectKey{Namespace: request.Namespace, Name: request.Name}, d)
	if err != nil {
		log.Info("failed to get deployment in namespace " + request.Namespace + " with name  " + d.Name)
		return reconcile.Result{}, nil
	}
	if d.Annotations == nil {
		d.Annotations = map[string]string{}
	}
	d.Annotations[domain.HeimdallLastChecked] = time.Now().Format(domain.TimeFormat)
	checked := []string{}
	for _, rep := range report {
		checked = append(checked, rep.ClusterImage.SHA256Path)
		if err := r.podService.LabelPods(&rep); err != nil {
			log.Error(err, "failed to label pod ")
			return reconcile.Result{}, nil
		}
	}
	d.Annotations[domain.HeimdallImagesChecked] = strings.Join(checked, ",")
	if err := r.client.Update(ctx, d); err != nil {
		log.Error(err, "failed to annotate deployment "+d.Namespace+" "+d.Name)
		return reconcile.Result{}, nil
	}
	return reconcile.Result{RequeueAfter: requeAfterFourHours}, nil
}

var _ reconcile.Reconciler = &ReconcileDeployment{}

// ReconcileImageMonitor reconciles a ImageMonitor object
type ReconcileDeployment struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	// turn into interfaces
	reportService *Reports
	podService    *cluster.Pods
	imageService  *cluster.ImageService
}
