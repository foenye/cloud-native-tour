package v2

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type FooCondition struct {
	Type   FooConditionType       `json:"type" protobuf:"bytes,1,opt,name=type,casttype=FooConditionType"`
	Status metav1.ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status,casttype=k8s.io/apimachinery/pkg/apis/meta/v1.ConditionStatus"`
}
type FooConditionType string

const (
	FooConditionTypeWorker FooConditionType = "Worker"
	FooConditionTypeConfig FooConditionType = "Config"
)
