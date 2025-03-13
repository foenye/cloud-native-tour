package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Group     = "hello.eonvon.github.io"
	Version   = "v1"
	Kind      = "Foo"
	Plural    = "foos"
	Singular  = "foo"
	ShortName = "fo"
	Name      = Plural + "." + Group
)

type FooSpec struct {
	// Msg say hello world!
	Msg string `json:"msg" protobuf:"bytes,1,opt,name=msg"`
	// Description provides some verbose information
	// +optional
	Description string `json:"description" protobuf:"bytes,2,opt,name=description"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Foo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec FooSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FooList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []Foo `json:"items" protobuf:"bytes,2,rep,name=items"`
}
