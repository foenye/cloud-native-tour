package core

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TestType is a top-level type. A client is created for it.
type TestType struct {
	metav1.TypeMeta
	// +optional
	metav1.ObjectMeta
	// +optional
	Status TestTypeStatus
}

// TestTypeList is a top-level list type. The client methods for lists are automatically created.
// You are not supposed to create a separated client for this one.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TestTypeList struct {
	metav1.TypeMeta
	// +optional
	metav1.ListMeta

	Items []TestType
}

type TestTypeStatus struct {
	Blah string
}
