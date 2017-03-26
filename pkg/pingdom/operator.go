package pingdom

import (
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/op/go-logging"
	pdom "github.com/russellcardullo/go-pingdom/pingdom"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/watch"
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

	eventCnt uint64

	ingInf cache.SharedIndexInformer
}

// New creates a new controller.
func New(namespace string, kclient *kubernetes.Clientset) *Operator {
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
	if !annotated(ing) {
		return
	}

	logp := fmt.Sprintf("handleAddIngress[%d]", atomic.AddUint64(&o.eventCnt, 1))
	log.Debugf("%s ingress=%+v", logp, ing)
	defer log.Debugf("%s end", logp)

	hosts := getIngressHosts(ing)
	if len(hosts) > 0 {
		err := o.createChecks(logp, ing, hosts)
		if err != nil {
			log.Errorf("%s error: %v", logp, err)
		}
	}
}

// Delete Pingdom checks if the ingress has the annotation.
func (o *Operator) handleDeleteIngress(obj interface{}) {
	ing := obj.(*v1beta1.Ingress)
	if !annotated(ing) {
		return
	}

	logp := fmt.Sprintf("handleDeleteIngress[%d]", atomic.AddUint64(&o.eventCnt, 1))
	log.Debugf("%s ingress=%+v", logp, ing)
	defer log.Debugf("%s end", logp)

	err := o.deleteChecks(logp, ing)
	if err != nil {
		log.Debugf("%s error: %v", logp, err)
	}
}

// Update Pingdom checks if the ingress has the annotation.
func (o *Operator) handleUpdateIngress(oldObj, newObj interface{}) {
	old, new := oldObj.(*v1beta1.Ingress), newObj.(*v1beta1.Ingress)
	if !annotated(new) {
		return
	}

	id := atomic.AddUint64(&o.eventCnt, 1)
	log.Debugf("handleUpdateIngress[%d] NOT YET IMPLEMENTED old=%+v new=%+v", id, old, new)
	defer log.Debugf("handleUpdateIngress[%d] end", id)
}

// Create a check for each host in the Ingress and annotates it
// with the checks metadata.
func (o *Operator) createChecks(logp string, ing *v1beta1.Ingress, hosts []string) error {
	phosts := make(map[string]int)

	for _, h := range hosts {
		id, err := o.createCheck(h)
		if err == nil {
			phosts[h] = id
			log.Debugf("%s added Pingdom check %d for host %s", logp, id, h)
		} else {
			log.Errorf("%s error: adding Pingdom check for host %s", logp, h)
		}
	}

	bytes, _ := json.Marshal(phosts)
	data := string(bytes)

	// Get a fresh copy of the ingress before updating.
	ing, err := o.kclient.Ingresses(ing.Namespace).Get(ing.Name)
	if err != nil {
		return fmt.Errorf("getting ingress: %v", err)
	}

	// Add annotation with the hosts and check IDs.
	ing.ObjectMeta.Annotations[checksAnnotation] = data

	_, err = o.kclient.Ingresses(ing.Namespace).Update(ing)
	if err != nil {
		return fmt.Errorf("updating ingress: %v", err)
	}

	return nil
}

// Delete all checks before the Ingress is deleted.
func (o *Operator) deleteChecks(logp string, ing *v1beta1.Ingress) error {
	data, ok := ing.ObjectMeta.Annotations[checksAnnotation]
	if !ok {
		return nil
	}

	var hosts map[string]int
	err := json.Unmarshal([]byte(data), &hosts)
	if err != nil {
		return fmt.Errorf("unmarshaling checks json: %v", err)
	}

	for host, id := range hosts {
		err := o.deleteCheck(id)
		if err == nil {
			log.Debugf("%s deleted check %d for host %s", logp, id, host)
		} else {
			log.Errorf("%s error deleting check %d for host %s", logp, id, host)
		}
	}

	return nil
}

func annotated(ing *v1beta1.Ingress) bool {
	_, ok := ing.ObjectMeta.Annotations[pingdomAnnotation]
	return ok
}

// Returns Ingress hosts
func getIngressHosts(ing *v1beta1.Ingress) []string {
	hosts := make([]string, 0)
	for _, r := range ing.Spec.Rules {
		if r.Host != "" {
			hosts = append(hosts, r.Host)
		}
	}
	return hosts
}
