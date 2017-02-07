package pingdom

import (
	"github.com/op/go-logging"
	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/1.5/pkg/watch"
	"k8s.io/client-go/1.5/rest"
)

var (
	log = logging.MustGetLogger("pingdom")
)

type Operator struct {
	kclient   *kubernetes.Clientset
	ingresses map[string]IngressChecks
}

type IngressChecks struct {
	Name  string
	Hosts map[string]int
}

// New creates a new controller.
func New(cfg *rest.Config) (*Operator, error) {
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	c := &Operator{
		kclient:   client,
		ingresses: make(map[string]IngressChecks),
	}

	return c, nil
}

// Run the controller.
func (c *Operator) Run(stopc <-chan struct{}) error {
	go func() {
		c.watchIngresses()
	}()

	return nil
}

func (c *Operator) watchIngresses() {
	w, err := c.kclient.Ingresses(api.NamespaceDefault).Watch(api.ListOptions{})
	if err != nil {
		log.Errorf("Error creating ingress watcher: %v", err)
	}

	for evt := range w.ResultChan() {
		et := watch.EventType(evt.Type)
		ing := evt.Object.(*v1beta1.Ingress)

		if et == watch.Added {
			log.Infof("Ingress %s added", ing.Name)
			c.addChecks(ing)
		}

		if et == watch.Modified {
			log.Infof("Ingress %s updated - NOT YET IMPLEMENTED", ing.Name)
		}

		if et == watch.Deleted {
			log.Infof("Ingress %s deleted", ing.Name)
			c.deleteChecks(ing)
		}
	}
}

func (c *Operator) addChecks(ing *v1beta1.Ingress) {
	ings := c.ingresses
	hosts := make(map[string]int)

	for _, r := range ing.Spec.Rules {
		log.Debugf("Adding Pingdom check for host %s", r.Host)
		hosts[r.Host] = 1
	}

	ic := IngressChecks{
		Name:  ing.Name,
		Hosts: hosts,
	}

	ings[ing.Name] = ic
	c.ingresses = ings
}

func (c *Operator) deleteChecks(ing *v1beta1.Ingress) {
	ings := c.ingresses

	if ic, ok := ings[ing.Name]; ok {
		ic = ings[ing.Name]

		for h, c := range ic.Hosts {
			log.Debugf("Deleting check %d for host %s", c, h)
		}

		delete(ings, ing.Name)
		c.ingresses = ings
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
