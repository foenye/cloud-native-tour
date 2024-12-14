/*
put install here, rather than in package hello
because hellov1 and hellov2 conversion import hello
*/

package install

import (
	helloyeahfogithubio "github.com/yeahfo/cloud-native-tour/api-aggregation-lib-multi-version/pkg/api/hello.yeahfo.github.io"
	helloyeahfogithubiov1 "github.com/yeahfo/cloud-native-tour/api-aggregation-lib-multi-version/pkg/api/hello.yeahfo.github.io/v1"
	helloyeahfogithubiov2 "github.com/yeahfo/cloud-native-tour/api-aggregation-lib-multi-version/pkg/api/hello.yeahfo.github.io/v2"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

// Install registers the API group and adds types to a scheme
func Install(scheme *runtime.Scheme) {
	utilruntime.Must(helloyeahfogithubio.AddToScheme(scheme))
	utilruntime.Must(helloyeahfogithubiov1.AddToScheme(scheme))
	utilruntime.Must(helloyeahfogithubiov2.AddToScheme(scheme))
	utilruntime.Must(scheme.SetVersionPriority(helloyeahfogithubiov2.SchemeGroupVersion, helloyeahfogithubiov1.SchemeGroupVersion))
}
