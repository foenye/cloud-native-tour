package v2

import (
	hellov2 "github.com/eonvon/cloud-native-tour/api/hello.eonvon.github.io/v2"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: hellov2.Group, Version: hellov2.Version}
var (
	// refer and use the SchemeBuilder in api/hello.eonvon.github.io/v2
	// as we need add default funcs, conversion funcs...
	localSchemeBuilder = &hellov2.SchemeBuilder
	AddToScheme        = localSchemeBuilder.AddToScheme
)
