package tpr

import (
	"sync/atomic"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

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
	store     *Store
	eventCnt  uint64
}

func New(namespace string, clientset kubernetes.Interface, store *Store) *Operator {
	return &Operator{
		tpr:       newTPR(clientset, tprKind, tprGroup, tprVersion, tprDescription, namespace),
		namespace: namespace,
		clientset: clientset,
		store:     store,
		eventCnt:  0,
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
			id := atomic.AddUint64(&o.eventCnt, 1)
			logger.Debugf("AddFunc[%d] check=%+v", id, check)
			defer logger.Debugf("AddFunc[%d] end", id)
			o.store.set(check)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			old, new := oldObj.(*PingdomCheck), newObj.(*PingdomCheck)
			id := atomic.AddUint64(&o.eventCnt, 1)
			logger.Debugf("UpdateFunc[%d]: old=%+v new=%+v", id, old, new)
			defer logger.Debugf("UpdateFunc[%d] end", id)
			o.store.set(new)
		},
		DeleteFunc: func(obj interface{}) {
			check := obj.(*PingdomCheck)
			id := atomic.AddUint64(&o.eventCnt, 1)
			logger.Debugf("DeleteFund[%d] check=%+v", id, check)
			defer logger.Debugf("DeleteFund[%d] end", id)
			o.store.delete(check)
		},
	})

	watcher.Run(stopCh)
	return nil
}

func (o *Operator) initResources() error {
	logger.Infof("creating TPR: %s", o.tpr.Name())
	err := o.tpr.CreateAndWait()
	if err == nil {
		logger.Infof("creating TPR: success")
	}
	return err
}
