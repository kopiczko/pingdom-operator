package pingdom

import (
	"github.com/op/go-logging"
	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/rest"
)

var (
	log = logging.MustGetLogger("pingdom")
)

type Operator struct {
	kclient *kubernetes.Clientset
}

// New creates a new controller.
func New(cfg *rest.Config) (*Operator, error) {
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	c := &Operator{
		kclient: client,
	}

	return c, nil
}

// Run the controller.
func (c *Operator) Run(stopc <-chan struct{}) error {
	log.Infof("Operator Run")
	return nil
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
