package v1

import (
	hellov1 "github.com/eonvon/cloud-native-tour/api/hello.eonvon.github.io/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var SchemeGroupVersion = schema.GroupVersion{Group: hellov1.Group, Version: hellov1.Version}

var (
	// refer and use the SchemeBuilder in api/hello.eonvon.github.io/v1
	// as we need add default funcs, conversion funcs...
	localSchemeBuilder = &hellov1.SchemeBuilder
	AddToScheme        = localSchemeBuilder.AddToScheme
)
