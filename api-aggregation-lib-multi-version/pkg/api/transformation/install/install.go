package install

import (
	"github.com/eonvon/cloud-native-tour/api-aggregation-lib-multi-version/pkg/api/transformation"
	transformationv1beta1 "github.com/eonvon/cloud-native-tour/api/transformation/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

func Install(scheme *runtime.Scheme) {
	utilruntime.Must(transformationv1beta1.Install(scheme))
	utilruntime.Must(transformation.AddToScheme(scheme))
	utilruntime.Must(scheme.SetVersionPriority(transformationv1beta1.SchemeGroupVersion))
}
