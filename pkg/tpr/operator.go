package tpr

import (
	"time"

	"k8s.io/client-go/kubernetes"

	"k8s.io/client-go/rest"

	logging "github.com/op/go-logging"
)

const (
	initRetryDelay = 10 * time.Second

	tprKind        = "check"
	tprGroup       = "pingdom.example.com"
	tprVersion     = "v1aplpha1"
	tprDescription = "Managed Pingdom uptime checks"
)

var (
	logger = logging.MustGetLogger("pingdom")
)

type Operator struct {
	tpr       *tpr
	namespace string
	clientset kubernetes.Interface
	config    *rest.Config
}

func New(namespace string, clientset kubernetes.Interface, config *rest.Config) *Operator {
	return &Operator{
		tpr:       newTPR(clientset, tprKind, tprGroup, tprVersion, tprDescription, namespace),
		namespace: namespace,
		clientset: clientset,
		config:    config,
	}
}

func (o *Operator) Run(stopc <-chan struct{}) error {
	for {
		err := o.initResources()
		if err == nil {
			break
		}
		logger.Errorf("Failed to init resources: %+v. retrying...", err)
		<-time.After(initRetryDelay)
	}
	return nil
}

func (o *Operator) initResources() error {
	logger.Infof("creating TPR: %s", o.tpr.Name())
	return o.tpr.CreateAndWait()
}
