package tpr

import (
	"fmt"
	"time"

	"github.com/rossf7/pingdom-operator/pkg/util"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
)

const (
	tprInitRetries    = 30
	tprInitRetryDelay = 3 * time.Second
)

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
