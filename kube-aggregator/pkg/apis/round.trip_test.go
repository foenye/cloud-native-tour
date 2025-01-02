package apis

import (
	"github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/install"
	apiregistrationv1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1"
	apiregistrationv1beta1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	"k8s.io/apimachinery/pkg/api/apitesting/roundtrip"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"testing"
)

func TestRoundTripToUnstructured(t *testing.T) {
	scheme := runtime.NewScheme()
	install.Install(scheme)

	roundtrip.RoundtripToUnstructured(t, scheme, fuzzer.MergeFuzzerFuncs(), sets.New(
		apiregistrationv1.SchemeGroupVersion.WithKind("CreateOptions"),
		apiregistrationv1.SchemeGroupVersion.WithKind("PatchOptions"),
		apiregistrationv1.SchemeGroupVersion.WithKind("UpdateOptions"),
		apiregistrationv1beta1.SchemeGroupVersion.WithKind("CreateOptions"),
		apiregistrationv1beta1.SchemeGroupVersion.WithKind("PatchOptions"),
		apiregistrationv1beta1.SchemeGroupVersion.WithKind("UpdateOptions"),
	), nil)
}
