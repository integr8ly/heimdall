package imagemonitor

import (
	"context"
	"github.com/integr8ly/heimdall/pkg/apis/imagemonitor/v1alpha1"
	"github.com/integr8ly/heimdall/pkg/cluster"
	"github.com/integr8ly/heimdall/pkg/domain"
	"github.com/pkg/errors"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

//whenever an image monitor resource is seen, the controller will check when a scan was last done
// if no scan has been done, it will trigger a scan by updating the labels on the deployment or deploymentconfig
// doing this will cause the deploymentconfig and deployment handlers to run the needed scans

var log = logf.Log.WithName("controller_imagemonitor")

const finalizer = "heimdall.rhmi.org"

func Add(mgr manager.Manager) error {
	client, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client")
	}

	return add(mgr, newReconciler(mgr, client))
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {

	c, err := controller.New("image-monitor-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}
	return c.Watch(&source.Kind{Type: &v1alpha1.ImageMonitor{}}, &handler.EnqueueRequestForObject{})
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, k8sClient kubernetes.Interface) reconcile.Reconciler {
	c := mgr.GetClient()
	r := &ReconcileImageMonitor{
		client:        c,
		objectLabeler: cluster.NewObjectLabeler(c),
	}
	return r
}

type ReconcileImageMonitor struct {
	client        client.Client
	objectLabeler *cluster.ObjectsLabeler
}

func (r *ReconcileImageMonitor) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// TODO need to add finalizer and remove finalizer on delete
	ctx := context.TODO()
	imageMon := &v1alpha1.ImageMonitor{}
	err := r.client.Get(context.TODO(), client.ObjectKey{Namespace: request.Namespace, Name: request.Name}, imageMon)
	if err != nil {
		if errors2.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	finalizers := imageMon.GetFinalizers()
	if imageMon.DeletionTimestamp != nil {
		if err := r.objectLabeler.RemoveLabelsAnnotations(ctx, map[string]string{domain.HeimdallMonitored: "true"}, imageMon.Namespace); err != nil {
			return reconcile.Result{}, err
		}
		for i, f := range finalizers {
			if f == finalizer {
				finalizers = append(finalizers[:i], finalizers[i+1:]...)
				imageMon.SetFinalizers(finalizers)
				break
			}
		}
		if err := r.client.Update(ctx, imageMon); err != nil {
			return reconcile.Result{}, errors.Wrap(err, "failed to remove finalizer")
		}
		return reconcile.Result{}, nil
	}

	// add our finalizer
	foundFinalizer := false
	for _, f := range finalizers {
		if f == finalizer {
			foundFinalizer = true
			break
		}
	}
	if !foundFinalizer {
		finalizers = append(finalizers, finalizer)
		imageMon.SetFinalizers(finalizers)
		if err := r.client.Update(ctx, imageMon); err != nil {
			return reconcile.Result{}, errors.Wrap(err, "failed to remove finalizer")
		}
	}
	log.Info("have image monitor for namespace " + imageMon.Namespace + " with name " + imageMon.Name)
	// find deployment configs and deployments that match in the namespace and label them
	if err := r.objectLabeler.LabelAllDeploymentsAndDeploymentConfigs(ctx, map[string]string{domain.HeimdallMonitored: "true"}, imageMon.Spec.ExcludePattern, imageMon.Namespace); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}
