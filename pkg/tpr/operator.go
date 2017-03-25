package tpr

import (
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"

	"k8s.io/client-go/rest"

	logging "github.com/op/go-logging"
)

const (
	initRetryDelay = 10 * time.Second

	tprInitRetries    = 30
	tprInitRetryDelay = 3 * time.Second
)

var (
	logger = logging.MustGetLogger("pingdom")
)

type Operator struct {
	Namespace string
	clientset kubernetes.Interface
	config    *rest.Config
}

func New(namespace string, clientset kubernetes.Interface, config *rest.Config) *Operator {
	return &Operator{
		Namespace: namespace,
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
	err := createTPR(o.clientset)
	if err != nil {
		return fmt.Errorf("failed to create TPR: %+v", err)
	}
	err = waitForTPRInit(o.clientset.CoreV1().RESTClient(), o.Namespace, tprInitRetries, tprInitRetryDelay)
	if err != nil {
		return fmt.Errorf("failed to wait for TPR: %+v", err)
	}
	return nil
}
