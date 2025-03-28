package install

import (
	"github.com/foenye/cloud-native-tour/crd-getting-started/pkg/apis/greeting"
	greetingv1 "github.com/foenye/cloud-native-tour/crd-getting-started/pkg/apis/greeting/v1"
	greetingv2 "github.com/foenye/cloud-native-tour/crd-getting-started/pkg/apis/greeting/v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

func Install(scheme *runtime.Scheme) {
	utilruntime.Must(greeting.AddToScheme(scheme))
	utilruntime.Must(greetingv1.Install(scheme))
	utilruntime.Must(greetingv2.Install(scheme))
	utilruntime.Must(scheme.SetVersionPriority(
		schema.GroupVersion{Group: greeting.GroupName, Version: greetingv2.GroupVersion.Version},
		schema.GroupVersion{Group: greeting.GroupName, Version: greetingv1.GroupVersion.Version}))
}
