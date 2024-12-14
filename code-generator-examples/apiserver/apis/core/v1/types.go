package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// TestType is a top-level type. A client is created for it.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TestType struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// +optional
	Status TestTypeStatus `json:"status,omitempty" protobuf:"bytes,2,opt,name=status"`
}

// TestTypeList is a top-level list type. The client methods for lists are automatically created.
// You are not supposed to create a separated client for this one.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TestTypeList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []TestType `json:"items" protobuf:"bytes,2,rep,name=items"`
}

type TestTypeStatus struct {
	Blah string `json:"blah" protobuf:"bytes,2,opt,name=blah"`
}
