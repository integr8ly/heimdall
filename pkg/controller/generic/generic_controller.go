package generic

import (
	"fmt"
	"strings"
	"time"

	"github.com/integr8ly/heimdall/pkg/cluster"
	"github.com/integr8ly/heimdall/pkg/controller/validation"
	"github.com/integr8ly/heimdall/pkg/domain"
	"github.com/integr8ly/heimdall/pkg/registry"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type logger interface {
	Error(err error, msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
}

// MakeGenericReconciler creates a generic reconciler that delegates the object
// access to an impl HeimdallObjectInterface
func MakeGenericReconciler(
	requeueInterval time.Duration,
	resourceName string,
	log logger,
	podService *cluster.Pods,
	clusterImageService *cluster.ImageService,
	registryImageService *registry.ImageService,
	impl HeimdallObjectInterface,
) *Reconciler {
	return &Reconciler{
		HeimdallObjectInterface: impl,

		requeueInterval: requeueInterval,
		resourceName:    resourceName,
		log:             log,
		reportService: &Reports{
			HeimdallObjectInterface: impl,
			resourceName:            resourceName,
			clusterImageService:     clusterImageService,
			registryImageService:    registryImageService,
		},
		podService: podService,
	}
}

// Reconciler is a generic implementation of a reconciler that delegates
// the object access logic to a HeimdallObjectInterface to check the object's
// images and label its pods
type Reconciler struct {
	HeimdallObjectInterface

	requeueInterval time.Duration
	resourceName    string
	log             logger

	reportService *Reports
	podService    *cluster.Pods
}

// HeimdallObjectInterface knows how to access resources watched by Heimdall
type HeimdallObjectInterface interface {
	GetObject(namespace, name string) (v1.Object, error)
	UpdateObject(v1.Object) error
	ListObjects(namespace string) ([]v1.Object, error)
	GetPodTemplateLabels(obj v1.Object) map[string]string
}

// Reconcile checks the images of an object managed by r's HeimdallObjectInterface
// and labels the pods accordingly
func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	obj, err := r.GetObject(request.Namespace, request.Name)
	if err != nil {
		r.log.Error(err, fmt.Sprintf("failed to get %s in namespace %s with name %s",
			r.resourceName, request.Name, request.Namespace))
		return reconcile.Result{}, err
	}

	if _, ok := obj.GetLabels()[domain.HeimdallMonitored]; !ok {
		return reconcile.Result{}, nil
	}

	images, err := r.reportService.GetImages(obj)
	if err != nil {
		return reconcile.Result{}, err
	}

	should, err := validation.ShouldCheck(obj, images)
	if err != nil && validation.IsParseErr(err) {
		delete(obj.GetAnnotations(), domain.HeimdallImagesChecked)
		if err := r.UpdateObject(obj); err != nil {
			r.log.Error(err, fmt.Sprintf(" failed to label %s %s/%s",
				r.resourceName,
				request.Namespace,
				request.Name,
			))
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, err
	}

	if !should {
		r.log.Info(fmt.Sprintf("criteria for re checking %s no met", request.Name))
		return reconcile.Result{}, nil
	}

	r.log.Info(fmt.Sprintf("%s %s in namespace %s is being monitored by heimdall",
		r.resourceName,
		request.Name,
		request.Namespace,
	))

	report, err := r.reportService.Generate(request.Namespace, request.Name)
	if err != nil {
		r.log.Error(err, "failed to generate a report for images in %s %s in namespace %s",
			r.resourceName,
			request.Name,
			request.Namespace,
		)
		return reconcile.Result{RequeueAfter: r.requeueInterval}, nil
	}

	r.log.Info(fmt.Sprintf("generated reports for %s", r.resourceName),
		"reports", len(report),
		"namespace", request.Namespace,
		"name", request.Name,
	)

	obj, err = r.GetObject(request.Namespace, request.Name)
	if err != nil {
		r.log.Info(fmt.Sprintf("failed to get %s in namespace %s with name %s",
			r.resourceName,
			request.Namespace,
			request.Name,
		))
		return reconcile.Result{}, nil
	}

	if obj.GetAnnotations() == nil {
		obj.SetAnnotations(map[string]string{})
	}
	annotations := obj.GetAnnotations()

	checked := []string{}
	for _, rep := range report {
		checked = append(checked, rep.ClusterImage.SHA256Path)
		if err := r.podService.LabelPods(&rep); err != nil {
			r.log.Error(err, "failed to label pod, will retry as soon as possible")
			return reconcile.Result{}, nil
		}
	}

	annotations[domain.HeimdallLastChecked] = time.Now().Format(domain.TimeFormat)
	annotations[domain.HeimdallImagesChecked] = strings.Join(checked, ",")

	if err := r.UpdateObject(obj); err != nil {
		r.log.Error(err, fmt.Sprintf("failed to annotate %s %s %s",
			r.resourceName,
			request.Namespace,
			request.Name,
		))
		return reconcile.Result{}, nil
	}

	return reconcile.Result{RequeueAfter: r.requeueInterval}, nil
}
