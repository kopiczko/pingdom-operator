package pingdom

import (
	"encoding/json"
	"os"
	"time"

	"github.com/op/go-logging"
	pdom "github.com/russellcardullo/go-pingdom/pingdom"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

var (
	log = logging.MustGetLogger("pingdom")
)

const (
	pingdomAnnotation = "monitoring.rossfairbanks.com/pingdom"
	checksAnnotation  = "monitoring.rossfairbanks.com/pingdom_checks"
	resyncPeriod      = 5 * time.Minute
)

type Operator struct {
	kclient *kubernetes.Clientset
	pclient *pdom.Client

	ingInf cache.SharedIndexInformer
}

// New creates a new controller.
func New(namespace string, kclient *kubernetes.Clientset, cfg *rest.Config) *Operator {
	pclient := pdom.NewClient(os.Getenv("PINGDOM_USER"), os.Getenv("PINGDOM_PASSWORD"), os.Getenv("PINGDOM_API_KEY"))

	c := &Operator{
		kclient: kclient,
		pclient: pclient,
	}

	ingress := kclient.Ingresses(namespace)

	c.ingInf = cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options api.ListOptions) (runtime.Object, error) {
				var v1Options v1.ListOptions
				v1.Convert_api_ListOptions_To_v1_ListOptions(&options, &v1Options, nil)
				return ingress.List(v1Options)
			},
			WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
				var v1Options v1.ListOptions
				v1.Convert_api_ListOptions_To_v1_ListOptions(&options, &v1Options, nil)
				return ingress.Watch(v1Options)
			},
		},
		&v1beta1.Ingress{}, resyncPeriod, cache.Indexers{},
	)

	c.ingInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleAddIngress,
		DeleteFunc: c.handleDeleteIngress,
		UpdateFunc: c.handleUpdateIngress,
	})

	return c
}

// Run the controller.
func (o *Operator) Run(stopc <-chan struct{}) error {
	go o.ingInf.Run(stopc)

	<-stopc
	return nil
}

// Create Pingdom checks if the ingress has the annotation.
func (o *Operator) handleAddIngress(obj interface{}) {
	ing := obj.(*v1beta1.Ingress)
	hosts := getIngressHosts(ing)

	if len(hosts) > 0 {
		o.createChecks(ing, hosts)
	}
}

// Delete Pingdom checks if the ingress has the annotation.
func (o *Operator) handleDeleteIngress(obj interface{}) {
	ing := obj.(*v1beta1.Ingress)

	if _, ok := ing.ObjectMeta.Annotations[pingdomAnnotation]; ok {
		o.deleteChecks(ing)
	}
}

// Update Pingdom checks if the ingress has the annotation.
func (o *Operator) handleUpdateIngress(old, cur interface{}) {
	ing := cur.(*v1beta1.Ingress)

	if _, ok := ing.ObjectMeta.Annotations[pingdomAnnotation]; ok {
		log.Infof("Updating ingress %s - NOT YET IMPLEMENTED", ing.Name)
	}
}

// Create a check for each host in the Ingress and annotates it
// with the checks metadata.
func (o *Operator) createChecks(ing *v1beta1.Ingress, hosts []string) error {
	phosts := make(map[string]int)

	for _, h := range hosts {
		id, err := o.createCheck(h)

		if err == nil {
			phosts[h] = id
			log.Debugf("Added Pingdom check %d for host %s", id, h)
		}
	}

	json, _ := json.Marshal(phosts)

	// Get a fresh copy of the ingress before updating.
	ing, err := o.kclient.Ingresses(ing.Namespace).Get(ing.Name)
	if err != nil {
		log.Errorf("Failed to get ingress: %v", err)
		return err
	}

	// Add annotation with the hosts and check IDs.
	an := ing.ObjectMeta.Annotations
	an[checksAnnotation] = string(json)

	ing.ObjectMeta.SetAnnotations(an)

	_, err = o.kclient.Ingresses(ing.Namespace).Update(ing)
	if err != nil {
		log.Errorf("Error updating ingress: %v", err)
	}

	return err
}

// Delete all checks before the Ingress is deleted.
func (o *Operator) deleteChecks(ing *v1beta1.Ingress) error {
	if data, ok := ing.ObjectMeta.Annotations[checksAnnotation]; ok {
		hosts := make(map[string]int)

		err := json.Unmarshal([]byte(data), &hosts)
		if err != nil {
			log.Errorf("Error unmarshaling checks json: %v", err)
			return err
		}

		for host, id := range hosts {
			err := o.deleteCheck(id)

			if err == nil {
				log.Debugf("Deleted check %d for host %s", id, host)
			}
		}
	}

	return nil
}

// Returns Ingress hosts if the Ingress has the Pingdom annotation.
func getIngressHosts(ing *v1beta1.Ingress) []string {
	hosts := make([]string, 0)

	if _, ok := ing.ObjectMeta.Annotations[pingdomAnnotation]; ok {
		for _, r := range ing.Spec.Rules {
			if r.Host != "" {
				hosts = append(hosts, r.Host)
			}
		}
	}

	return hosts
}
