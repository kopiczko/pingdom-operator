package pingdom

import (
	"encoding/json"
	"os"
	"time"

	"github.com/op/go-logging"
	pdom "github.com/russellcardullo/go-pingdom/pingdom"

	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/1.5/pkg/runtime"
	"k8s.io/client-go/1.5/pkg/watch"
	"k8s.io/client-go/1.5/rest"
	"k8s.io/client-go/1.5/tools/cache"
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

type IngressChecks struct {
	Hosts map[string]int
}

// New creates a new controller.
func New(cfg *rest.Config) (*Operator, error) {
	kclient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	pclient := pdom.NewClient(os.Getenv("PINGDOM_USER"), os.Getenv("PINGDOM_PASSWORD"), os.Getenv("PINGDOM_API_KEY"))

	c := &Operator{
		kclient: kclient,
		pclient: pclient,
	}

	c.ingInf = cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options api.ListOptions) (runtime.Object, error) {
				return kclient.Ingresses(api.NamespaceAll).List(options)
			},
			WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
				return kclient.Ingresses(api.NamespaceAll).Watch(options)
			},
		},
		&v1beta1.Ingress{}, resyncPeriod, cache.Indexers{},
	)

	c.ingInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleAddIngress,
		DeleteFunc: c.handleDeleteIngress,
		UpdateFunc: c.handleUpdateIngress,
	})

	return c, nil
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

	if _, ok := ing.ObjectMeta.Annotations[pingdomAnnotation]; ok {
		o.createChecks(ing)
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

func (o *Operator) createChecks(ing *v1beta1.Ingress) error {
	hosts := make(map[string]int)

	for _, r := range ing.Spec.Rules {
		if r.Host != "" {
			id, err := o.createCheck(r.Host)

			if err == nil {
				hosts[r.Host] = id
				log.Debugf("Added Pingdom check %d for host %s", id, r.Host)
			}
		}
	}

	json, _ := json.Marshal(hosts)

	// Get a fresh copy of the ingress before updating.
	ing, err := o.kclient.Ingresses(ing.Namespace).Get(ing.Name)
	if err != nil {
		log.Errorf("Failed to get ingress: %v", err)
		return err
	}

	an := ing.ObjectMeta.Annotations
	an[checksAnnotation] = string(json)

	ing.ObjectMeta.SetAnnotations(an)

	_, err = o.kclient.Ingresses(ing.Namespace).Update(ing)
	if err != nil {
		log.Errorf("Error updating ingress: %v", err)
	}

	return err
}

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
