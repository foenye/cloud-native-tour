package install

import (
	"github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration"
	"github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1"
	"github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

// Install registers the API group and adds types to a scheme
func Install(scheme *runtime.Scheme) {
	utilruntime.Must(apiregistration.AddToScheme(scheme))
	utilruntime.Must(v1.Install(scheme))
	utilruntime.Must(v1beta1.Install(scheme))
	utilruntime.Must(scheme.SetVersionPriority(v1.SchemeGroupVersion, v1beta1.SchemeGroupVersion))
}
