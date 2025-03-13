package disallow

import (
	"context"
	"fmt"
	"github.com/eonvon/cloud-native-tour/api-aggregation-lib-multi-version/pkg/api/hello.eonvon.github.io"
	"io"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission"
)

var _ admission.ValidationInterface = &Disallow{}

const PluginDisallowFoo = "DisallowFoo"

type Disallow struct {
	admission.Handler
}

// Register disallowed Foo admission plugin.
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginDisallowFoo, func(config io.Reader) (admission.Interface, error) {
		return &Disallow{Handler: *admission.NewHandler(admission.Create)}, nil
	})
}

func (fooDisallow *Disallow) Validate(_ context.Context, attrs admission.Attributes, _ admission.ObjectInterfaces) (err error) {

	if attrs.GetKind().GroupKind() != hello.SchemeGroupVersion.WithKind("Disallow").GroupKind() {
		return nil
	}

	metaAccessor, err := meta.Accessor(attrs.GetObject())
	if err != nil {
		return err
	}
	namespace := metaAccessor.GetNamespace()
	if namespace == metav1.NamespaceSystem {
		return errors.NewForbidden(attrs.GetResource().GroupResource(),
			fmt.Sprintf("%s/%s", attrs.GetNamespace(), attrs.GetName()),
			fmt.Errorf("namespace/%s is not permitted, please change the resource namespace", namespace))
	}

	return nil
}
