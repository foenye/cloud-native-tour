package foo

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/eonvon/cloud-native-tour/api-aggregation-lib-multi-version/pkg/api/hello.eonvon.github.io"
	"github.com/eonvon/cloud-native-tour/api-aggregation-lib-multi-version/pkg/api/transformation"
	transformationv1beta1 "github.com/eonvon/cloud-native-tour/api/transformation/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"strings"
)

type getter interface {
	rest.Getter
}

var (
	_                rest.Storage                  = &Base64REST{}
	_                rest.NamedCreater             = &Base64REST{}
	_                rest.GroupVersionKindProvider = &Base64REST{}
	groupVersionKind                               = schema.GroupVersionKind{
		Group:   transformationv1beta1.GroupVersion.Group,
		Version: transformationv1beta1.GroupVersion.Version,
		Kind:    "Base64",
	}
)

type Base64REST struct {
	fooGetter getter
	scheme    runtime.Scheme
}

func NewBase64REST(fooGetter getter, scheme *runtime.Scheme) *Base64REST {
	return &Base64REST{fooGetter: fooGetter, scheme: *scheme}
}

func (base64REST *Base64REST) GroupVersionKind(schema.GroupVersion) schema.GroupVersionKind {
	return groupVersionKind
}

func (base64REST *Base64REST) Create(ctx context.Context, name string, obj runtime.Object,
	_ rest.ValidateObjectFunc, _ *metav1.CreateOptions) (runtime.Object, error) {
	command := obj.(*transformation.Base64)

	// Get the namespace form the context (populated from the URL).
	namespace, exists := request.NamespaceFrom(ctx)
	if !exists {
		return nil, errors.NewBadRequest("namespace is required")
	}

	info, exists := request.RequestInfoFrom(ctx)
	if !exists {
		return nil, errors.NewBadRequest("request info is required")
	}

	// require name/namespace in the body to match URL if specified
	if len(command.Name) > 0 && command.Name != name {
		errs := field.ErrorList{field.Invalid(field.NewPath("metadata").Child("name"), command.Name,
			"must match the foo name if specified")}
		return nil, errors.NewInvalid(groupVersionKind.GroupKind(), name, errs)
	}
	if len(command.Namespace) > 0 && command.Namespace != namespace {
		errs := field.ErrorList{field.Invalid(field.NewPath("metadata").Child("namespace"), command.Namespace,
			"must match the foo namespace if specified")}
		return nil, errors.NewInvalid(groupVersionKind.GroupKind(), name, errs)
	}

	// Lookup foo
	fooObj, err := base64REST.fooGetter.Get(ctx, name, &metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	foo := fooObj.(*hello.Foo)
	groupVersions := schema.GroupVersions([]schema.GroupVersion{{Group: info.APIGroup, Version: info.APIVersion}})
	out := unstructured.Unstructured{}
	if err := base64REST.scheme.Convert(foo, &out, groupVersions); err != nil {
		return nil, errors.NewBadRequest(fmt.Sprintf("unknown request version %s, %s", groupVersions, err))
	}

	jsonBytes, err := out.MarshalJSON()
	if err != nil {
		return nil, errors.NewInternalError(err)
	}

	if len(command.Spec.FieldPath) > 0 {
		fields := strings.Split(command.Spec.FieldPath, ".")
		mapObj, copied, err := unstructured.NestedFieldNoCopy(out.Object, fields...)
		if !copied || err != nil {
			return nil, errors.NewBadRequest(fmt.Sprintf("undefined field path %s", command.Spec.FieldPath))
		}
		jsonBytes, err = json.Marshal(mapObj)
		if err != nil {
			return nil, errors.NewBadRequest(fmt.Sprintf("unexpected nested object %s", err))
		}
	}

	represent := command.DeepCopy()
	represent.Status = transformation.Base64Status{Output: base64.StdEncoding.EncodeToString(jsonBytes)}
	return represent, nil
}

func (base64REST *Base64REST) New() runtime.Object {
	return &transformation.Base64{}
}

// Destroy cleans up resources on shutdown.
func (base64REST *Base64REST) Destroy() {
}
