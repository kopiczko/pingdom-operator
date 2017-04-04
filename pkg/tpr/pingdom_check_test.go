package tpr

import (
	"encoding/json"
	"testing"

	"k8s.io/client-go/pkg/api/unversioned"

	"github.com/stretchr/testify/assert"
)

var data = `{
	"kind":"CheckList","items":[],
	"metadata":{"selfLink":"/apis/example.com/v1alpha1/testkinds","resourceVersion":"319773"},
	"apiVersion":"example.com/v1alpha1"
}`

func TestPingdomCheckUnmarshal(t *testing.T) {
	want := &PingdomCheckList{
		TypeMeta: unversioned.TypeMeta{Kind: "CheckList", APIVersion: "example.com/v1alpha1"},
		ListMeta: unversioned.ListMeta{SelfLink: "/apis/example.com/v1alpha1/testkinds", ResourceVersion: "319773"},
		Items:    []*PingdomCheck{},
	}
	v := new(PingdomCheckList)
	err := json.Unmarshal([]byte(data), v)
	assert.Nil(t, err)
	assert.Equal(t, want, v)
}
