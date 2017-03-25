package tpr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/pkg/api/v1"
)

func newClientset(nodes int) *fake.Clientset {
	clientset := fake.NewSimpleClientset()
	for i := 0; i < nodes; i++ {
		n := &v1.Node{}
		n.Name = fmt.Sprintf("node%d", i)
		clientset.CoreV1().Nodes().Create(n)
	}
	return clientset
}

func TestCreateTPR(t *testing.T) {
	clientset := newClientset(3)

	tpr, err := clientset.ExtensionsV1beta1().ThirdPartyResources().List(v1.ListOptions{})
	assert.Equal(t, 0, len(tpr.Items))

	err = createTPR(clientset)
	assert.Nil(t, err)

	tpr, err = clientset.ExtensionsV1beta1().ThirdPartyResources().List(v1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(tpr.Items))
	assert.Equal(t, "check.pingdom.example.com", tpr.Items[0].Name)
	assert.Equal(t, tprDescription, tpr.Items[0].Description)
}
