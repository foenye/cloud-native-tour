package apiregistration

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// APIService represents a server for a particular GroupVersion.
// Name must be "version.group".
type APIService struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	// Spec contains information for locating and communicating with a server.
	Spec APIServiceSpec
	// Status contains derived information about an API server.
	Status APIServiceStatus
}

// APIServiceSpec contains information for locating and communicating with a server.
// Only https is supported, though you are able to disable certificate verification.
type APIServiceSpec struct {
	// Service is reference to the service for this API server. It must communicate on port 443.
	// If the Service is null, that means the handling for the API groupversion is handled locally on this server.
	// The call will simply delegate to the normal handler chain to be fulfilled.
	// +optional
	Service *ServiceReference
	// Group is the API group name this server hosts.
	Group string
	// Version is the API version this server hosts. For example, "v1"
	Version string
	// InsecureSkipTLSVerify disables TLS certificate verification when communicating with this server.
	// This is strongly discouraged. You should use the CABundle instanced.
	InsecureSkipTLSVerify bool
	// CABundle is a PEM encoded CA bundle which will be used to validate an API server's serving certificate.
	// If unspecified, system trust roots on the apiserver are use.
	// +listType=atomic
	// +optional
	CABundle []byte
	// GroupPriorityMinimum is the priority this group should hava at least. Higher priority means that the group is
	// preferred by clients to lower priority ones.
	// Note that other versions of this group might specify even higher GroupPriorityMinimum values such that the whole
	// group gets a higher priority.
	// The primary sort is based on GroupPriorityMinimum, ordered highest number to lowest (20 before 10).
	// The secondary sort is based on the alphabetical comparison of the name of the object. (v1.bar before v1.foo).
	// We'd recommend something like: *.k8s.io (except extensions) at 18000 and PaaSes (Openshift, Deis) are recommended
	// to be in the 2000s.
	GroupPriorityMinimum int32
	// VersionPriority controls the ordering of this API version inside of this group. Must be greater zero.
	// The primary sort based on VersionPriority, ordered highest to lowest (20 before 10).
	// Since it's inside a group, the number can be small, probably in the 10s.
	// In case of equal version priorities, the version string will be used to compute the order inside a group.
	// If the version string is "kube-like", it will sort above non "kube-like" version strings, which are ordered
	// lexicographically. "kube-like" versions start with a "v", then are followed by a number (the major version),
	// then optionally the string "alpha" or "beta" and another number (the minor version). These are sort first by
	// GA > bate > alpha (where GA is a version with no suffix such as beta or alpha), and the comparing major version,
	// then minor version. An example sorted list of versions:
	// v10, v2, v1, v11beta2, v10beta1, v12alpha1, v11alpha2, foo1, foo10.
	VersionPriority int32
}

// APIServiceStatus contains derived information about an API server.
type APIServiceStatus struct {
	// Conditions is current service state of apiService.
	// +listType=map
	// +listMapKey=type
	Conditions []APIServiceCondition
}

// APIServiceConditionType is a valid value for APIServiceCondition.Type
type APIServiceConditionType string

const (
	// Available indicates that the service exists and is reachable.
	Available APIServiceConditionType = "Available"
)

type ConditionStatus string

// These are valid condition statuses.
// "ConditionTrue" means a resource is in the condition;
// "ConditionFalse" means a resource is not in the condition;
// "ConditionUnknown" means kubernetes can't decide if a resource is in the condition or not.
// In the future, we could add other intermediate conditions, e.g. ConditionDegraded.
const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)

// APIServiceCondition describes conditions for an APIService.
type APIServiceCondition struct {
	// Type is the type of the condition.
	Type APIServiceConditionType
	// Status is the status of the condition, can be True, False, Unknown.
	Status ConditionStatus
	// LastTransitionTime is the Last time of the condition transitioned form status to another.
	LastTransitionTime metav1.Time
	// Reason unique, one-word, format CamelCase reason for the condition's last transition.
	Reason string
	// Message should be human-readable message indicating details about last transition.
	Message string
}

// ServiceReference holds a reference to Service.legacy.k8s.io
type ServiceReference struct {
	// Namespace is the namespace of the service.
	Namespace string
	// Name is the name of the service.
	Name string
	// Port if specified, the port on the service that hosting the service.
	// Default to 443 for backward compatibility.
	// `port` should be a valid port number (1-65535, inclusive)
	// +optional
	Port int32
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// APIServiceList is a list of APIService object.
type APIServiceList struct {
	metav1.TypeMeta
	metav1.ListMeta

	Items []APIService
}
