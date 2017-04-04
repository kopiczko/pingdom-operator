package pingdom

import (
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/op/go-logging"
	"github.com/rossf7/pingdom-operator/pkg/tpr"
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
	store   *tpr.Store
	eventc  chan interface{}

	checks *pingdomChecks

	eventCnt uint64

	ingInf cache.SharedIndexInformer
}

// New creates a new controller.
func New(namespace string, kclient *kubernetes.Clientset, store *tpr.Store) *Operator {
	pclient := pdom.NewClient(os.Getenv("PINGDOM_USER"), os.Getenv("PINGDOM_PASSWORD"), os.Getenv("PINGDOM_API_KEY"))

	c := &Operator{
		kclient: kclient,
		pclient: pclient,
		store:   store,
		eventc:  make(chan interface{}),
		checks:  newPingdomChecks(),
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

	c.store.Handler = tpr.StoreEventHandlerFuncs{
		SetFunc: func(namespace, name string, spec tpr.Spec) {
			c.eventc <- setCheckSpecEvent{Namespace: namespace, Name: name, Check: spec}
		},
		DeleteFunc: func(namespace, name string, spec tpr.Spec) {
			c.eventc <- deleteCheckSpecEvent{Namespace: namespace, Name: name, Check: spec}
		},
	}

	c.ingInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.eventc <- addIngressEvent{obj: obj}
		},
		UpdateFunc: func(old, new interface{}) {
			c.eventc <- updateIngressEvent{old: old, new: new}
		},
		DeleteFunc: func(obj interface{}) {
			c.eventc <- deleteIngressEvent{obj: obj}
		},
	})

	return c
}

// Run the controller.
func (o *Operator) Run(stopc <-chan struct{}) error {
	go o.ingInf.Run(stopc)
	go o.run()

	<-stopc
	close(o.eventc)
	return nil
}

func (o *Operator) run() {
	for e := range o.eventc {
		switch e := e.(type) {
		case addIngressEvent:
			o.handleAddIngress(e.Ingress())
		case deleteIngressEvent:
			o.handleDeleteIngress(e.Ingress())
		case updateIngressEvent:
			o.handleUpdateIngress(e.OldIngress(), e.NewIngress())
		case setCheckSpecEvent:
			o.handleSetCheckSpec(e.Namespace, e.Name, e.Check)
		case deleteCheckSpecEvent:
			o.handleDeleteCheckSpec(e.Namespace, e.Name, e.Check)
		default:
			log.Error("Unhandled event: %+v", e)
		}
	}
}

// Create Pingdom checks if the ingress has the annotation.
func (o *Operator) handleAddIngress(ing *v1beta1.Ingress) {
	checkName, ok := annotation(ing)
	if !ok {
		return
	}
	if hasChecks(ing) {
		return
	}

	logp := fmt.Sprintf("AddIngress[%d]", atomic.AddUint64(&o.eventCnt, 1))
	log.Debugf("%s obj=%s", logp, ing.Name)
	defer log.Debugf("%s end", logp)

	hosts := getIngressHosts(ing)
	if len(hosts) == 0 {
		return
	}

	checkSpec, ok := o.store.Get(ing.Namespace, checkName)
	if !ok {
		checkSpec = defaultCheckSpec
	}

	err := o.createChecks(logp, ing, hosts, checkName, checkSpec)
	if err != nil {
		log.Errorf("%s error: %v", logp, err)
	}
}

// Delete Pingdom checks if the ingress has the annotation.
func (o *Operator) handleDeleteIngress(ing *v1beta1.Ingress) {
	checkName, ok := annotation(ing)
	if !ok {
		return
	}

	logp := fmt.Sprintf("DeleteIngress[%d]", atomic.AddUint64(&o.eventCnt, 1))
	log.Debugf("%s obj=%s", logp, ing.Name)
	defer log.Debugf("%s end", logp)

	err := o.deleteChecks(logp, ing, checkName)
	if err != nil {
		log.Errorf("%s error: %v", logp, err)
	}
}

// Update Pingdom checks if the ingress has the annotation.
func (o *Operator) handleUpdateIngress(old, new *v1beta1.Ingress) {
	// TODO at least remove checks if new is not annotated and pass checks otherwise
	if _, ok := annotation(new); !ok {
		return
	}

	logp := fmt.Sprintf("UpdateIngress[%d]", atomic.AddUint64(&o.eventCnt, 1))
	log.Debugf("%s NOT YET IMPLEMENTED old=%s new=%s", logp, old.Name, new.Name)
	defer log.Debugf("%s end", logp)
}

func (o *Operator) handleSetCheckSpec(namespace, name string, checkSpec tpr.Spec) {
	logp := fmt.Sprintf("SetCheckSpec[%d]", atomic.AddUint64(&o.eventCnt, 1))
	log.Debugf("%s namespace=%s name=%s", logp, namespace, name)
	defer log.Debugf("%s end", logp)

	for _, id := range o.checks.Get(name) {
		err := o.updateCheck(id, checkSpec)
		if err == nil {
			log.Debugf("%s updated checkID=%d", logp, id)
		} else {
			log.Debugf("%s error updating checkID=%d:", logp, id, err)
		}
	}
}

func (o *Operator) handleDeleteCheckSpec(namespace, name string, checkSpec tpr.Spec) {
	logp := fmt.Sprintf("DeleteCheckSpec[%d]", atomic.AddUint64(&o.eventCnt, 1))
	log.Debugf("%s namespace=%s name=%s", logp, namespace, name)
	defer log.Debugf("%s end", logp)

	for _, id := range o.checks.Get(name) {
		err := o.updateCheck(id, defaultCheckSpec)
		if err == nil {
			log.Debugf("%s setting default checkID=%d", logp, id)
		} else {
			log.Debugf("%s error setting default checkID=%d: %v", logp, id, err)
		}
	}
}

// Create a check for each host in the Ingress and annotates it
// with the checks metadata.
func (o *Operator) createChecks(logp string, ing *v1beta1.Ingress, hosts []string, checkName string, checkSpec tpr.Spec) error {
	phosts := make(map[string]int)

	for _, h := range hosts {
		id, err := o.createCheck(h, checkSpec)
		if err == nil {
			phosts[h] = id
			o.checks.Add(checkName, id)
			log.Debugf("%s added Pingdom check %d for host %s", logp, id, h)
		} else {
			log.Errorf("%s error: adding Pingdom check for host %s: %v", logp, h, err)
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
func (o *Operator) deleteChecks(logp string, ing *v1beta1.Ingress, checkName string) error {
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
			o.checks.Delete(checkName, id)
			log.Debugf("%s deleted check %d for host %s", logp, id, host)
		} else {
			log.Errorf("%s error deleting check %d for host %s", logp, id, host)
		}
	}

	return nil
}

func annotation(ing *v1beta1.Ingress) (v string, ok bool) {
	v, ok = ing.ObjectMeta.Annotations[pingdomAnnotation]
	return
}

func hasChecks(ing *v1beta1.Ingress) bool {
	v, _ := ing.ObjectMeta.Annotations[checksAnnotation]
	return len(v) > 0
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

type addIngressEvent struct {
	obj interface{}
}

func (e addIngressEvent) Ingress() *v1beta1.Ingress { return e.obj.(*v1beta1.Ingress) }

type deleteIngressEvent struct {
	obj interface{}
}

func (e deleteIngressEvent) Ingress() *v1beta1.Ingress { return e.obj.(*v1beta1.Ingress) }

type updateIngressEvent struct {
	old, new interface{}
}

func (e updateIngressEvent) OldIngress() *v1beta1.Ingress { return e.old.(*v1beta1.Ingress) }
func (e updateIngressEvent) NewIngress() *v1beta1.Ingress { return e.new.(*v1beta1.Ingress) }

type setCheckSpecEvent struct {
	Namespace, Name string
	Check           tpr.Spec
}

type deleteCheckSpecEvent struct {
	Namespace, Name string
	Check           tpr.Spec
}
