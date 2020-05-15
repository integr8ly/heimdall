package statefulset

import (
	"time"

	"github.com/integr8ly/heimdall/pkg/cluster"
	"github.com/integr8ly/heimdall/pkg/controller/generic"
	"github.com/integr8ly/heimdall/pkg/registry"
	"github.com/integr8ly/heimdall/pkg/rhcc"
	imagesv1 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	"github.com/pkg/errors"
	v12 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_statefulset")

const labelFormat = "heimdall.%s"

const requeueInterval = time.Hour * 4

// Add creates a new StatefulSet Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	client, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client")
	}

	isClient, err := imagesv1.NewForConfig(mgr.GetConfig())
	if err != nil {
		return errors.Wrap(err, "failed to create images client")
	}

	return add(mgr, newReconciler(mgr, client, isClient))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, client kubernetes.Interface, isClient *imagesv1.ImageV1Client) reconcile.Reconciler {
	clusterImageService := cluster.NewImageService(client, isClient)
	registryImageService := registry.NewImagesService(&registry.Client{}, &rhcc.Client{}, &rhcc.Client{})

	impl := &objectInterface{
		client: client.AppsV1(),
	}

	return generic.MakeGenericReconciler(
		requeueInterval,
		"stateful set",
		log,
		cluster.NewPods(mgr.GetClient()),
		clusterImageService,
		registryImageService,
		impl,
	)
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("statefulset-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource StatefulSet
	return c.Watch(&source.Kind{Type: &v12.StatefulSet{}}, &handler.EnqueueRequestForObject{})
}

// blank assignment to verify that ReconcileStatefulSet implements HeimdallObjectInterface
var _ generic.HeimdallObjectInterface = &objectInterface{}

// objectInterface is an implementation of generic.HeimdallObjectInterface
// that knows how to access stateful sets
type objectInterface struct {
	client v1.AppsV1Interface
}

// ListObjects gets the stateful sets in namespace as v1.Object types
func (r *objectInterface) ListObjects(namespace string) ([]metav1.Object, error) {
	statefulSets, err := r.client.StatefulSets(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]metav1.Object, len(statefulSets.Items))
	for i, s := range statefulSets.Items {
		result[i] = &s
	}

	return result, nil
}

// GetPodTemplateLabels gets the pod template labels for a given stateful set obj
func (r *objectInterface) GetPodTemplateLabels(obj metav1.Object) map[string]string {
	return obj.(*v12.StatefulSet).Spec.Template.Labels
}

// GetObject gets a stateful set name in namespace
func (r *objectInterface) GetObject(namespace, name string) (metav1.Object, error) {
	return r.client.StatefulSets(namespace).Get(name, metav1.GetOptions{})
}

// UpdateObject updates a stateful set
func (r *objectInterface) UpdateObject(obj metav1.Object) error {
	_, err := r.client.StatefulSets(obj.GetNamespace()).Update(obj.(*v12.StatefulSet))
	return err
}
