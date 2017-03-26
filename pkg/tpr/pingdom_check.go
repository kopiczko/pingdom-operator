package tpr

import (
	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/runtime"
)

type Spec struct {
	RetryInterval int `json:"retryInterval"`
}

/*
	All code below is boilerplate to make TPR watching functionality work.
*/

type pingdomCheckFuncs struct{}

func (pingdomCheckFuncs) NewObject() runtime.Object     { return new(PingdomCheck) }
func (pingdomCheckFuncs) NewObjectList() runtime.Object { return new(PingdomCheckList) }

type PingdomCheck struct {
	unversioned.TypeMeta `json:",inline"`
	v1.ObjectMeta        `json:"metadata,omitempty"`

	Spec Spec `json:"spec"`
}

type PingdomCheckList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`

	Items []*PingdomCheck `json:"items"`
}
