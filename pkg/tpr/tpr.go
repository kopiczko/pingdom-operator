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
	tprKind        = "check"
	tprGroup       = "pingdom.example.com"
	tprVersion     = "v1aplpha1"
	tprDescription = "Managed Pingdom uptime checks"
)

func tprName() string {
	return fmt.Sprintf("%s.%s", tprKind, tprGroup)
}

func createTPR(clientset kubernetes.Interface) error {
	logger.Infof("creating TPR: %s", tprName())
	tpr := &v1beta1.ThirdPartyResource{
		ObjectMeta: v1.ObjectMeta{
			Name: tprName(),
		},
		Versions: []v1beta1.APIVersion{
			{Name: tprVersion},
		},
		Description: tprDescription,
	}
	_, err := clientset.ExtensionsV1beta1().ThirdPartyResources().Create(tpr)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("creating TPR failed: %+v", err)
		}
	}

	return nil
}

func waitForTPRInit(restcli rest.Interface, ns string, maxRetries int, retryDelay time.Duration) error {
	uri := fmt.Sprintf("/apis/%s/%s/namespaces/%s/%ss", tprGroup, tprVersion, ns, tprKind)
	return util.Retry(retryDelay, maxRetries, func() (bool, error) {
		_, err := restcli.Get().RequestURI(uri).DoRaw()
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}
