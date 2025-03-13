package apiservice

import (
	"context"
	"fmt"
	"github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration"
	"github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/validation"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
)

type apiServerStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// apiServerStrategy must implements rest.RESTCreateUpdateStrategy
// rest.CreateUpdateResetFieldsStrategy is a union of rest.RESTCreateUpdateStrategy and ResetFieldsStrategy
var _ rest.CreateUpdateResetFieldsStrategy = Strategy{}

type Strategy = apiServerStrategy

// NewStrategy creates a new apiServerStrategy.
func NewStrategy(typer runtime.ObjectTyper) rest.CreateUpdateResetFieldsStrategy {
	return Strategy{typer, names.SimpleNameGenerator}
}

func (Strategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		apiregistration.GroupName + "/v1":      fieldpath.NewSet(fieldpath.MakePathOrDie("status")),
		apiregistration.GroupName + "/v1beta1": fieldpath.NewSet(fieldpath.MakePathOrDie("status")),
	}
}

func (Strategy) NamespaceScoped() bool {
	return false
}

func (Strategy) PrepareForCreate(_ context.Context, obj runtime.Object) {
	apiService := obj.(*apiregistration.APIService)
	apiService.Status = apiregistration.APIServiceStatus{}

	// mark local API services as immediately available on create
	if apiService.Spec.Service == nil {
		apiregistration.SetAPIServiceCondition(apiService, apiregistration.NewLocalAvailableAPIServiceCondition())
	}
}

func (Strategy) Validate(_ context.Context, obj runtime.Object) field.ErrorList {
	return validation.ValidateAPIService(obj.(*apiregistration.APIService))
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (Strategy) WarningsOnCreate(_ context.Context, _ runtime.Object) []string {
	return nil
}

func (Strategy) Canonicalize(_ runtime.Object) {
}

func (Strategy) AllowCreateOnUpdate() bool {
	return false
}

func (Strategy) PrepareForUpdate(_ context.Context, obj, old runtime.Object) {
	obj.(*apiregistration.APIService).Status = old.(*apiregistration.APIService).Status
}

func (Strategy) ValidateUpdate(_ context.Context, obj, old runtime.Object) field.ErrorList {
	return validation.ValidateAPIServiceUpdate(obj.(*apiregistration.APIService), old.(*apiregistration.APIService))
}

// WarningsOnUpdate returns warnings for the given update.
func (Strategy) WarningsOnUpdate(_ context.Context, _, _ runtime.Object) []string {
	return nil
}

func (Strategy) AllowUnconditionalUpdate() bool {
	return false
}

type apiServerStatusStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

var _ rest.UpdateResetFieldsStrategy = StatusStrategy{}

type StatusStrategy apiServerStatusStrategy

// NewStatusStrategy creates a new StatusStrategy.
func NewStatusStrategy(typer runtime.ObjectTyper) rest.UpdateResetFieldsStrategy {
	return StatusStrategy{typer, names.SimpleNameGenerator}
}

func (StatusStrategy) NamespaceScoped() bool {
	return false
}

func (StatusStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (StatusStrategy) PrepareForUpdate(_ context.Context, obj, old runtime.Object) {
	newAPIService := obj.(*apiregistration.APIService)
	oldAPIService := old.(*apiregistration.APIService)
	newAPIService.Spec = oldAPIService.Spec
	newAPIService.Labels = oldAPIService.Labels
	newAPIService.Annotations = oldAPIService.Annotations
	newAPIService.Finalizers = oldAPIService.Finalizers
	newAPIService.OwnerReferences = oldAPIService.OwnerReferences
}

// ValidateUpdate validates an update of StatusStrategy.
func (StatusStrategy) ValidateUpdate(_ context.Context, obj, old runtime.Object) field.ErrorList {
	return validation.ValidateAPIServiceStatusUpdate(obj.(*apiregistration.APIService), old.(*apiregistration.APIService))
}

// WarningsOnUpdate returns warning for the given update.
func (StatusStrategy) WarningsOnUpdate(_ context.Context, _, _ runtime.Object) []string {
	return nil
}

// Canonicalize normalizes the object after validation.
func (StatusStrategy) Canonicalize(_ runtime.Object) {

}

func (StatusStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (StatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		apiregistration.GroupName + "/v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
			fieldpath.MakePathOrDie("metadata"),
		),
		apiregistration.GroupName + "/v1bate1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
			fieldpath.MakePathOrDie("metadata"),
		),
	}
}

// GetAttrs returns the labels and fields of an API server for filtering purposes.
func GetAttrs(object runtime.Object) (labels.Set, fields.Set, error) {

	if apiService, casted := object.(*apiregistration.APIService); !casted {
		return nil, nil, fmt.Errorf("given object is no a APIService")
	} else {
		return apiService.ObjectMeta.Labels, generic.ObjectMetaFieldsSet(&apiService.ObjectMeta, true), nil
	}
}

// MatchAPIService is the filter used by generic etcd backend to watch events from etcd to clients of the apiserver
// only interested in specific labels/fields.
func MatchAPIService(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}
