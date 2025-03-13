/*
put install here, rather than in package hello
because hellov1 and hellov2 conversion import hello
*/

package install

import (
	"github.com/eonvon/cloud-native-tour/api-aggregation-lib-multi-version/pkg/api/hello.eonvon.github.io"
	hellov1 "github.com/eonvon/cloud-native-tour/api-aggregation-lib-multi-version/pkg/api/hello.eonvon.github.io/v1"
	hellov2 "github.com/eonvon/cloud-native-tour/api-aggregation-lib-multi-version/pkg/api/hello.eonvon.github.io/v2"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

// Install registers the API group and adds types to a scheme
func Install(scheme *runtime.Scheme) {
	utilruntime.Must(hello.AddToScheme(scheme))
	utilruntime.Must(hellov1.AddToScheme(scheme))
	utilruntime.Must(hellov2.AddToScheme(scheme))
	utilruntime.Must(scheme.SetVersionPriority(hellov2.SchemeGroupVersion, hellov1.SchemeGroupVersion))
}
