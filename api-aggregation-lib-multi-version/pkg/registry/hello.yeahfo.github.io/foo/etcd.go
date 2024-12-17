package foo

import (
	"context"
	"fmt"
	"github.com/yeahfo/cloud-native-tour/api-aggregation-lib-multi-version/pkg/api/hello.yeahfo.github.io"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
)

var _ rest.ShortNamesProvider = &REST{}

type REST struct {
	*genericregistry.Store
}

func (R *REST) ShortNames() []string {
	return []string{hello.ShortName}
}

type Storage struct {
	Foo    *REST
	Config *ConfigREST
	Status *StatusREST
	Base64 *Base64REST
}

// NewREST returns a REST Storage object that will work against API services.
func NewREST(scheme *runtime.Scheme, optionsGetter generic.RESTOptionsGetter) (*Storage, error) {
	strategy := newStrategy(scheme)
	statusStrategy := newStatusStrategy(strategy)

	store := &genericregistry.Store{
		NewFunc:                   func() runtime.Object { return &hello.Foo{} },
		NewListFunc:               func() runtime.Object { return &hello.FooList{} },
		PredicateFunc:             matchFoo,
		DefaultQualifiedResource:  hello.Resource(hello.Plural),
		SingularQualifiedResource: hello.Resource(hello.Singular),

		CreateStrategy: strategy,
		UpdateStrategy: strategy,
		DeleteStrategy: strategy,
		TableConvertor: strategy,
	}

	options := &generic.StoreOptions{RESTOptions: optionsGetter, AttrFunc: getAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		return nil, err
	}

	configStore := *store
	statusStore := *store
	statusStore.UpdateStrategy = statusStrategy
	statusStore.ResetFieldsStrategy = statusStrategy

	return &Storage{Foo: &REST{Store: store}, Config: &ConfigREST{Store: &configStore},
		Status: &StatusREST{Store: &statusStore}}, nil
}

var _ rest.Patcher = &ConfigREST{}
var _ rest.ResetFieldsStrategy = &ConfigREST{}

// ConfigREST implements the config subresource for a Foo.
type ConfigREST struct {
	Store *genericregistry.Store
}

// GetResetFields implements rest.ResetFieldsStrategy.
func (configREST *ConfigREST) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return configREST.GetResetFields()
}

func (configREST *ConfigREST) Destroy() {
	// Given that underlying store is shared with REST,
	// we don't destroy it here explicitly.
}

// Get retrieves the object form storage. Its required to support Patch.
func (configREST *ConfigREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	fooObj, err := configREST.Store.Get(ctx, name, options)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFound(hello.Resource("foos/config"), name)
		}
		return nil, err
	}
	foo := fooObj.(*hello.Foo)
	return configFromFoo(foo), nil
}

// New creates a new Config resource.
func (configREST *ConfigREST) New() runtime.Object {
	return &hello.Config{}
}

var _ rest.UpdatedObjectInfo = &configUpdatedObjectInfo{}

type configUpdatedObjectInfo struct {
	name           string
	requestObjInfo rest.UpdatedObjectInfo
}

func (root *configUpdatedObjectInfo) Preconditions() *metav1.Preconditions {
	return root.requestObjInfo.Preconditions()
}

func (root *configUpdatedObjectInfo) UpdatedObject(ctx context.Context, oldObj runtime.Object) (newObj runtime.Object, err error) {
	foo, ok := oldObj.DeepCopyObject().(*hello.Foo)
	if !ok {
		return nil, errors.NewBadRequest(fmt.Sprintf("expected existing object type to be Foo, got %T", foo))
	}

	// if zero-value, the existing object does not exist
	if len(foo.ResourceVersion) == 0 {
		return nil, errors.NewNotFound(hello.Resource(hello.Plural+"/config"), root.name)
	}

	oldConfig := configFromFoo(foo)

	// old config -> new config
	newConfig, err := root.requestObjInfo.UpdatedObject(ctx, oldConfig)
	if err != nil {
		return nil, err
	}
	if newConfig == nil {
		return nil, errors.NewBadRequest("nil update parse to Config")
	}

	config, casted := newConfig.(*hello.Config)
	if !casted {
		return nil, errors.NewBadRequest(fmt.Sprintf("expected input object type to Config, but %T", newConfig))
	}

	// validate precondition if specified (resourceVersion matching is handled by storage).
	if len(config.UID) > 0 && config.UID != foo.UID {
		return nil, errors.NewConflict(hello.Resource(hello.Plural+"/config"), root.name,
			fmt.Errorf("precondition failed: UID in preconditaion: %v, UID in object meta: %v", config.UID, foo.UID))
	}

	// move fields to object and return
	foo.Spec.Config.Msg = config.Spec.Msg
	foo.Spec.Config.Description = config.Spec.Description
	foo.ResourceVersion = config.ResourceVersion
	return foo, nil
}

// Update alters the spec.config subset of an object.
// Normally option createValidation an option updateValidation are validating admission control funcs.
//
//	see https://github.com/kubernetes/kubernetes/blob/d25c0a1bdb81b7a9b52abf10687d701c82704602/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/patch.go#L270
//	see https://github.com/kubernetes/kubernetes/blob/d25c0a1bdb81b7a9b52abf10687d701c82704602/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/update.go#L210-L216
func (configREST *ConfigREST) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo,
	createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, _ bool,
	options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	obj, _, err := configREST.Store.Update(ctx, name,
		&configUpdatedObjectInfo{name, objInfo},
		toConfigCreateValidation(createValidation),
		toConfigUpdateValidation(updateValidation),
		false,
		options,
	)
	if err != nil {
		return nil, false, err
	}
	foo := obj.(*hello.Foo)
	newConfig := configFromFoo(foo)
	return newConfig, false, nil
}

func toConfigUpdateValidation(validation rest.ValidateObjectUpdateFunc) rest.ValidateObjectUpdateFunc {
	return func(ctx context.Context, obj, old runtime.Object) error {
		config := configFromFoo(obj.(*hello.Foo))
		oldConfig := configFromFoo(old.(*hello.Foo))
		return validation(ctx, config, oldConfig)
	}
}

func toConfigCreateValidation(validation rest.ValidateObjectFunc) rest.ValidateObjectFunc {
	return func(ctx context.Context, obj runtime.Object) error {
		config := configFromFoo(obj.(*hello.Foo))
		return validation(ctx, config)
	}
}

// configFromFoo returns a config subresource form a Foo.
func configFromFoo(foo *hello.Foo) runtime.Object {
	return &hello.Config{
		ObjectMeta: metav1.ObjectMeta{
			Name:              foo.Name,
			Namespace:         foo.Namespace,
			UID:               foo.UID,
			ResourceVersion:   foo.ResourceVersion,
			CreationTimestamp: foo.CreationTimestamp,
		},
		Spec: hello.ConfigSpec{
			Msg:         foo.Spec.Config.Msg,
			Description: foo.Spec.Config.Description,
		},
	}
}

var (
	_ getter                   = &StatusREST{}
	_ rest.Storage             = &StatusREST{}
	_ rest.Updater             = &StatusREST{}
	_ rest.TableConvertor      = &StatusREST{}
	_ rest.ResetFieldsStrategy = &StatusREST{}
)

// StatusREST implements the REST endpoint for changing the status of a Foo.
type StatusREST struct {
	Store *genericregistry.Store
}

// GetResetFields implements rest.ResetFieldsStrategy
func (statusREST *StatusREST) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return statusREST.Store.GetResetFields()
}

func (statusREST *StatusREST) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return statusREST.Store.ConvertToTable(ctx, object, tableOptions)
}

// Update alters the status subset of an object.
func (statusREST *StatusREST) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo,
	createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, _ bool,
	options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	// We are explicitly setting forceAllowCreate to false in the call to the underlying storage because
	// subresource should never allow create on update.
	return statusREST.Store.Update(ctx, name, objInfo, createValidation, updateValidation, false, options)
}

// New creates a Foo resource.
func (statusREST *StatusREST) New() runtime.Object {
	return &hello.Foo{}
}

// Destroy cleans up resources on shutdown.
func (statusREST *StatusREST) Destroy() {
}

// Get retrieves the object from the storage. It is required to support Patch.
func (statusREST *StatusREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return statusREST.Store.Get(ctx, name, options)
}
