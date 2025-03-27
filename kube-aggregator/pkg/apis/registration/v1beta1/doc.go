// +k8s:deepcopy-gen=package
// +k8s:protobuf-gen=package
// +k8s:conversion-gen=github.com/foenye/cloud-native-tour/kube-aggregator/pkg/apis/registration
// +k8s:openapi-gen=true
// +groupName=registration.foen.ye
// +k8s:defaulter-gen=TypeMeta
// +k8s:prerelease-lifecycle-gen=true

// Package v1beta1 contains the API Registration API, which is responsible for
// registering an API `Group`/`Version` with another kubernetes like API server.
// The `APIService` holds information about the other API server in
// `APIServiceSpec` type as well as general `TypeMeta` and `ObjectMeta`. The
// `APIServiceSpec` type have the main configuration needed to do the
// aggregation. Any request coming for specified `Group`/`Version` will be
// directed to the service defined by `ServiceReference` (on port 443) after
// validating the target using provided `CABundle` or skipping validation
// if development flag `InsecureSkipTLSVerify` is set. `Priority` is controlling
// the order of this API group in the overall discovery document.
// The return status is a set of conditions for this aggregation. Currently
// there is only one condition named "Available", if true, it means the
// api/server requests will be redirected to specified API server.
package v1beta1
