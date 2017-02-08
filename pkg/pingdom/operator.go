package pingdom

import (
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
	resyncPeriod = 5 * time.Minute
)

type Operator struct {
	kclient *kubernetes.Clientset
	pclient *pdom.Client

	ingInf    cache.SharedIndexInformer
	ingresses map[string]IngressChecks
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
		kclient:   kclient,
		pclient:   pclient,
		ingresses: make(map[string]IngressChecks),
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
	})

	return c, nil
}

// Run the controller.
func (o *Operator) Run(stopc <-chan struct{}) error {
	go o.ingInf.Run(stopc)

	<-stopc
	return nil
}

func (o *Operator) handleAddIngress(obj interface{}) {
	ing := obj.(*v1beta1.Ingress)
	ings := o.ingresses

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

	ic := IngressChecks{
		Hosts: hosts,
	}

	ings[ing.Name] = ic
	o.ingresses = ings
}

func (o *Operator) handleDeleteIngress(obj interface{}) {
	ing := obj.(*v1beta1.Ingress)
	ings := o.ingresses

	if ic, ok := ings[ing.Name]; ok {
		ic = ings[ing.Name]

		for host, id := range ic.Hosts {
			err := o.deleteCheck(id)

			if err == nil {
				delete(ings, ing.Name)
				o.ingresses = ings

				log.Debugf("Deleted check %d for host %s", id, host)
			}
		}
	}
}

/*

import (
	"github.com/rossf7/pingdom-operator/pkg/k8sutil"

	"github.com/op/go-logging"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/1.5/kubernetes"
	extensionsobj "k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/1.5/rest"
	"k8s.io/client-go/1.5/tools/cache"
)

const (
	tprPingdom = "pingdom." + v1alpha1.TPRGroup
)
*/

/*
// Run the controller.
func (c *Operator) Run(stopc <-chan struct{}) error {
	errChan := make(chan error)
	go func() {
		if err := c.createTPRs(); err != nil {
			errChan <- err
			return
		}
		errChan <- nil
	}()

	select {
	case err := <-errChan:
		if err != nil {
			return err
		}

		log.Info("TPR API endpoints ready")
	case <-stopc:
		return nil
	}

	<-stopc
	return nil
}

// Creates third party resources.
func (c *Operator) createTPRs() error {
	tprs := []*extensionsobj.ThirdPartyResource{
		{
			ObjectMeta: apimetav1.ObjectMeta{
				Name: tprPingdom,
			},
			Versions: []extensionsobj.APIVersion{
				{Name: v1alpha1.TPRVersion},
			},
			Description: "",
		},
	}

	tprClient := c.kclient.Extensions().ThirdPartyResources()

	for _, tpr := range tprs {
		if _, err := tprClient.Create(tpr); err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}

		log.Infof("TPR created: %s", tpr.Name)
	}

	// We have to wait for the TPRs to be ready. Otherwise the initial watch may fail.
	return k8sutil.WaitForTPRReady(c.kclient.CoreV1().RESTClient(), v1alpha1.TPRGroup, v1alpha1.TPRVersion, v1alpha1.TPRPingdomName)
}
*/
