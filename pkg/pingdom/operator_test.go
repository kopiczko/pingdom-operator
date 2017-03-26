package pingdom

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func TestGetIngressHostsWithAnnotation(t *testing.T) {
	ing := v1beta1.Ingress{
		ObjectMeta: v1.ObjectMeta{
			Annotations: map[string]string{
				"monitoring.rossfairbanks.com/pingdom": "test",
			},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				v1beta1.IngressRule{Host: "test.example.com"},
				v1beta1.IngressRule{Host: "test.example.org"},
			},
		},
	}

	hosts := getIngressHosts(&ing)

	assert.Equal(t, 2, len(hosts))
	assert.Equal(t, "test.example.com", hosts[0])
	assert.Equal(t, "test.example.org", hosts[1])
}
