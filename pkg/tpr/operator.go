package tpr

import (
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"k8s.io/client-go/rest"

	logging "github.com/op/go-logging"
)

const (
	initRetryDelay = 10 * time.Second

	tprKind        = "check"
	tprGroup       = "pingdom.example.com"
	tprVersion     = "v1aplpha1"
	tprDescription = "Managed Pingdom uptime checks for Ingress hosts"
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

func (o *Operator) Run(stopCh <-chan struct{}) error {
	for {
		err := o.initResources()
		if err == nil {
			break
		}
		logger.Errorf("Failed to init resources: %+v. retrying...", err)
		<-time.After(initRetryDelay)
	}

	watcher := o.tpr.Watcher(pingdomCheckFuncs{}, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			check := obj.(*PingdomCheck)
			logger.Debugf("AddFund: %+v", check)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			old, new := oldObj.(*PingdomCheck), newObj.(*PingdomCheck)
			logger.Debugf("UpdateFunc: old:%+v new:%+v", old, new)
		},
		DeleteFunc: func(obj interface{}) {
			check := obj.(*PingdomCheck)
			logger.Debugf("DeleteFund: %+v", check)
		},
	})

	watcher.Run(stopCh)
	return nil
}

func (o *Operator) initResources() error {
	logger.Infof("creating TPR: %s", o.tpr.Name())
	return o.tpr.CreateAndWait()
}
