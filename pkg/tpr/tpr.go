package tpr

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/rossf7/pingdom-operator/pkg/util"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const (
	tprInitRetries    = 30
	tprInitRetryDelay = 3 * time.Second

	// Period or re-synchronizing the list of objects in k8s watcher.
	// 0 means that re-sync will be delayed as long as possible, until the
	// watch will be closed or timed out.
	resyncPeriod time.Duration = 0
)

// zeroObjectFuncs provides zero values of an object and objects' list ready to
// be decoded. The provided zero values must not be reused by zeroObjectFuncs.
type zeroObjectFuncs interface {
	NewObject() runtime.Object
	NewObjectList() runtime.Object
}

type watcher struct {
	informer *cache.Controller
}

func (w watcher) Run(stopCh <-chan struct{}) { w.informer.Run(stopCh) }

type tpr struct {
	clientset kubernetes.Interface
	rest      rest.Interface
	namespace string

	kind        string
	group       string
	version     string
	description string

	name string

	endpointList  string
	endpointWatch string

	listWatch     *cache.ListWatch
	listWatchOnce sync.Once // listWatch guard
}

func newTPR(clientset kubernetes.Interface, kind, group, version, description, namespace string) *tpr {
	if len(namespace) > 0 {
		namespace = "/namespaces/" + namespace
	}
	return &tpr{
		clientset:     clientset,
		rest:          clientset.CoreV1().RESTClient(),
		namespace:     namespace,
		kind:          kind,
		group:         group,
		version:       version,
		description:   description,
		name:          fmt.Sprintf("%s.%s", tprKind, tprGroup),
		endpointList:  fmt.Sprintf("/apis/%s/%s%s/%ss", group, version, namespace, kind),
		endpointWatch: fmt.Sprintf("/apis/%s/%s%s/watch/%ss", group, version, namespace, kind),
	}
}

func (t *tpr) Name() string { return t.name }

// CreateAndWait create a TPR and waits till it is initialized in the cluster.
func (t *tpr) CreateAndWait() error {
	err := t.create()
	if err != nil {
		fmt.Errorf("creating TPR: %+v", err)
	}
	err = t.waitInit()
	if err != nil {
		fmt.Errorf("waiting TPR initialization: %+v", err)
	}
	return nil
}

// Watcher creates a watcher streaming this TPR events to handler. All event
// objects are created using provided zeroObjectFuncs.
func (t *tpr) Watcher(obj zeroObjectFuncs, handler cache.ResourceEventHandler) watcher {
	// initialize ListWatch once; could be done in newTPR but feels good to
	// have that code in a single function, as ListWatch should produce
	// objects with the type handler consumes
	t.listWatchOnce.Do(func() {
		t.listWatch = &cache.ListWatch{
			ListFunc: func(options api.ListOptions) (runtime.Object, error) {
				req := t.rest.Get().AbsPath(t.endpointList)
				data, err := req.DoRaw()
				if err != nil {
					return nil, err
				}
				v := obj.NewObjectList()
				if err := json.Unmarshal(data, v); err != nil {
					return nil, err
				}
				return v, nil
			},
			WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
				req := t.rest.Get().AbsPath(t.endpointWatch)
				stream, err := req.Stream()
				if err != nil {
					return nil, err
				}
				watcher := watch.NewStreamWatcher(&decoder{
					stream: stream,
					obj:    obj,
				})
				return watcher, nil
			},
		}
	})

	_, informer := cache.NewInformer(t.listWatch, obj.NewObject(), resyncPeriod, handler)
	return watcher{informer: informer}
}

// create is extracted for testing because fake REST client does not work.
// Therefore waitInit can not be tested.
func (t *tpr) create() error {
	tpr := &v1beta1.ThirdPartyResource{
		ObjectMeta: v1.ObjectMeta{
			Name: t.name,
		},
		Versions: []v1beta1.APIVersion{
			{Name: t.version},
		},
		Description: t.description,
	}
	_, err := t.clientset.ExtensionsV1beta1().ThirdPartyResources().Create(tpr)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}

func (t *tpr) waitInit() error {
	return util.Retry(tprInitRetryDelay, tprInitRetries, func() (bool, error) {
		_, err := t.rest.Get().RequestURI(t.endpointList).DoRaw()
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

type decoder struct {
	stream io.ReadCloser
	obj    zeroObjectFuncs
}

func (d *decoder) Decode() (action watch.EventType, object runtime.Object, err error) {
	dec := json.NewDecoder(d.stream)
	var e struct {
		Type   watch.EventType
		Object runtime.Object
	}
	e.Object = d.obj.NewObject()
	if err := dec.Decode(&e); err != nil {
		return watch.Error, nil, err
	}
	return e.Type, e.Object, nil
}

func (d *decoder) Close() {
	d.stream.Close()
}
