package pingdom

import (
	"testing"

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

	if len(hosts) == 2 {
		if hosts[0] != "test.example.com" {
			t.Errorf("Expected host not found %s", hosts[0])
		}

		if hosts[1] != "test.example.org" {
			t.Errorf("Expected host not found %s", hosts[1])
		}

	} else {
		t.Errorf("Expected 2 hosts but found %d", len(hosts))
	}
}

func TestGetIngressHostsWithoutAnnotation(t *testing.T) {
	ing := v1beta1.Ingress{
		ObjectMeta: v1.ObjectMeta{
			Annotations: map[string]string{
				"example.com/test": "test",
			},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				v1beta1.IngressRule{Host: "test.example.com"},
			},
		},
	}

	hosts := getIngressHosts(&ing)

	if len(hosts) != 0 {
		t.Errorf("Expected 0 hosts but found %d", len(hosts))
	}
}
