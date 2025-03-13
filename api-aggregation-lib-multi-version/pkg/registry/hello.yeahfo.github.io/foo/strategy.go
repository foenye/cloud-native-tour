package foo

import (
	"context"
	"fmt"
	"github.com/eonvon/cloud-native-tour/api-aggregation-lib-multi-version/pkg/api/hello.eonvon.github.io"
	"github.com/eonvon/cloud-native-tour/api-aggregation-lib-multi-version/pkg/api/hello.eonvon.github.io/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
	"time"
)

// strategy interfaces implementations
var (
	_ rest.TableConvertor           = strategy{}
	_ rest.ResetFieldsStrategy      = strategy{}
	_ rest.RESTCreateUpdateStrategy = strategy{}
)

type strategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// newStrategy creates and returns a foo strategy instance.
func newStrategy(objectTyper runtime.ObjectTyper) strategy {
	return strategy{ObjectTyper: objectTyper, NameGenerator: names.SimpleNameGenerator}
}

// NamespaceScoped implements rest.RESTCreateUpdateStrategy
func (strategy) NamespaceScoped() bool {
	return true
}

// PrepareForCreate implements rest.RESTCreateUpdateStrategy
func (strategy) PrepareForCreate(_ context.Context, obj runtime.Object) {
	foo := obj.(*hello.Foo)
	foo.Status = hello.FooStatus{Phase: hello.FooPhaseProcessing}
}

// Validate implements rest.RESTCreateUpdateStrategy
func (strategy) Validate(_ context.Context, obj runtime.Object) field.ErrorList {
	foo := obj.(*hello.Foo)
	return validation.ValidateFoo(foo)
}

// WarningsOnCreate implements rest.RESTCreateUpdateStrategy
// returns warnings for the creation of the given object.
func (strategy) WarningsOnCreate(_ context.Context, _ runtime.Object) []string {
	return nil
}

// Canonicalize implements rest.RESTCreateUpdateStrategy
func (strategy) Canonicalize(_ runtime.Object) {
}

// AllowCreateOnUpdate implements rest.RESTCreateUpdateStrategy
func (strategy) AllowCreateOnUpdate() bool {
	return false
}

// PrepareForUpdate implements rest.RESTCreateUpdateStrategy
func (strategy) PrepareForUpdate(_ context.Context, _, _ runtime.Object) {
}

// ValidateUpdate implements rest.RESTCreateUpdateStrategy
func (strategy) ValidateUpdate(_ context.Context, _, _ runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// WarningsOnUpdate implements rest.RESTCreateUpdateStrategy
// // returns warnings for the given update.
func (strategy) WarningsOnUpdate(_ context.Context, _, _ runtime.Object) []string {
	return nil
}

// AllowUnconditionalUpdate implements rest.RESTCreateUpdateStrategy
func (strategy) AllowUnconditionalUpdate() bool {
	return false
}

// GetResetFields implements rest.ResetFieldsStrategy
// returns the set of fields that get reset by the strategy and should not be modified by the user.
// (only do reset in put/patch actions, not for create action)
func (strategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		hello.GroupName + "/v2": fieldpath.NewSet(fieldpath.MakePathOrDie("status")),
	}
}

// ConvertToTable implements rest.TableConvertor
func (strategy) ConvertToTable(_ context.Context, object runtime.Object, _ runtime.Object) (*metav1.Table, error) {
	var table metav1.Table

	table.ColumnDefinitions = []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Status", Type: "string", Format: "status", Description: "status of where the Foo is in its lifecycle"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		{Name: "Message", Type: "string", Format: "message", Description: "foo message", Priority: 1},              // kubectl -o wide
		{Name: "Description", Type: "string", Format: "description", Description: "foo message plus", Priority: 1}, // kubectl -o wide
	}

	switch typed := object.(type) {
	case *hello.Foo:
		table.ResourceVersion = typed.ResourceVersion
		addSomeFooToTable(&table, *typed)
	case *hello.FooList:
		table.ResourceVersion = typed.ResourceVersion
		table.Continue = typed.Continue
		addSomeFooToTable(&table, typed.Items...)
	default:
	}
	return &table, nil
}

func addSomeFooToTable(table *metav1.Table, someFoo ...hello.Foo) {
	for _, foo := range someFoo {
		ageColumnRepresent := "<unknown>"
		if creationTimestamp := foo.CreationTimestamp; !creationTimestamp.IsZero() {
			ageColumnRepresent = duration.HumanDuration(time.Since(creationTimestamp.Time))
		}
		table.Rows = append(table.Rows, metav1.TableRow{
			//						Name	|Status				|Age				|Message			|Description
			Cells:  []interface{}{foo.Name, foo.Status.Phase, ageColumnRepresent, foo.Spec.Config.Msg, foo.Spec.Config.Description},
			Object: runtime.RawExtension{Object: &foo},
		})
	}
}

type statusStrategy struct {
	strategy
}

func newStatusStrategy(strategy strategy) *statusStrategy {
	return &statusStrategy{strategy: strategy}
}

// GetResetFields implements rest.ResetFieldsStrategy returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (statusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		hello.GroupName + "/v2": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
			fieldpath.MakePathOrDie("metadata", "deletionTimestamp"),
			fieldpath.MakePathOrDie("metadata", "ownerReferences"),
		),
	}
}

// PrepareForUpdate implements rest.RESTCreateUpdateStrategy
func (statusStrategy) PrepareForUpdate(_ context.Context, obj, old runtime.Object) {
	newFoo := obj.(*hello.Foo)
	oldFoo := old.(*hello.Foo)
	newFoo.Spec = oldFoo.Spec
	newFoo.DeletionTimestamp = nil
	newFoo.OwnerReferences = oldFoo.OwnerReferences
}

// ValidateUpdate implements rest.RESTCreateUpdateStrategy
func (statusStrategy) ValidateUpdate(_ context.Context, _, _ runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// WarningsOnUpdate implements rest.RESTCreateUpdateStrategy
// returns warnings for the given update.
func (statusStrategy) WarningsOnUpdate(_ context.Context, _, _ runtime.Object) []string {
	return nil
}

// matchFoo is the filter used by the generic etcd backend to watch events from etcd to clients of the apiserver
// only interested in specific labels/fields.
func matchFoo(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: getAttrs,
	}
}

// getAttrs returns labels.Set, fields.Set, and error in case the given runtime.Object is not a Foo
func getAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	foo, casted := obj.(*hello.Foo)
	if !casted {
		return nil, nil, fmt.Errorf("given object is not a Foo")
	}
	return foo.ObjectMeta.Labels, generic.ObjectMetaFieldsSet(&foo.ObjectMeta, true), nil
}
