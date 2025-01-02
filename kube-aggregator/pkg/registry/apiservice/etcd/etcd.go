package etcd

import (
	"context"
	"fmt"
	"github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration"
	"github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/registry/apiservice"
	"k8s.io/apimachinery/pkg/api/meta"
	metatable "k8s.io/apimachinery/pkg/api/meta/table"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
)

// REST implements a RESTStorage for API services against etcd.
type REST struct {
	*genericregistry.Store
}

// NewREST returns a RESTStorage object that will work against API services.
func NewREST(scheme *runtime.Scheme, restOptionsGetter generic.RESTOptionsGetter) *REST {
	strategy := apiservice.NewStrategy(scheme)
	store := &genericregistry.Store{
		NewFunc:                   func() runtime.Object { return &apiregistration.APIService{} },
		NewListFunc:               func() runtime.Object { return &apiregistration.APIService{} },
		PredicateFunc:             apiservice.MatchAPIService,
		DefaultQualifiedResource:  apiregistration.Resource("apiservices"),
		SingularQualifiedResource: apiregistration.Resource("apiservice"),

		CreateStrategy:      strategy,
		UpdateStrategy:      strategy,
		DeleteStrategy:      strategy,
		ResetFieldsStrategy: strategy,

		// TODO: define table converter that exposes more than name/creation timestamp.
		TableConvertor: rest.NewDefaultTableConvertor(apiregistration.Resource("apiservices")),
	}
	storeOptions := &generic.StoreOptions{RESTOptions: restOptionsGetter, AttrFunc: apiservice.GetAttrs}
	if err := store.CompleteWithOptions(storeOptions); err != nil {
		panic(err)
	}
	return &REST{store}
}

// REST implements rest.CategoriesProvider
var _ rest.CategoriesProvider = &REST{}

// Categories implements the rest.CategoriesProvider interface. Returns a list of categories a resource is part of.
func (REST) Categories() []string {

	return []string{"api-extensions"}
}

// ConvertToTable implements the rest.TableConvertor interface for REST
func (REST) ConvertToTable(_ context.Context, object runtime.Object, _ runtime.Object) (*metav1.Table, error) {
	table := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
			{Name: "Service", Type: "string", Description: "The reference to the service that hosts the API endpoint."},
			{Name: "Available", Type: "string", Description: "Whether this service is available."},
			{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		},
	}

	if listAccessor, err := meta.ListAccessor(object); err == nil {
		table.ResourceVersion = listAccessor.GetResourceVersion()
		table.Continue = listAccessor.GetContinue()
		table.RemainingItemCount = listAccessor.GetRemainingItemCount()
	} else {
		if commonAccessor, err := meta.CommonAccessor(object); err == nil {
			table.ResourceVersion = commonAccessor.GetResourceVersion()
		}
	}

	var err error
	table.Rows, err = metatable.MetaToTableRow(object, func(obj runtime.Object, m metav1.Object, name, age string) ([]interface{}, error) {
		svc := obj.(*apiregistration.APIService)
		service := "Local"
		if svc.Spec.Service != nil {
			service = fmt.Sprintf("%s/%s", svc.Spec.Service.Namespace, svc.Spec.Service.Name)
		}
		status := string(apiregistration.ConditionUnknown)
		if condition := getCondition(svc.Status.Conditions, apiregistration.Available); condition != nil {
			switch {
			case condition.Status == apiregistration.ConditionTrue:
				status = string(condition.Status)
			case len(condition.Reason) > 0:
				status = fmt.Sprintf("%s (%s)", condition.Status, condition.Reason)
			default:
				status = string(condition.Status)
			}
		}
		return []interface{}{name, service, status, age}, nil
	})
	return table, err
}

func getCondition(conditions []apiregistration.APIServiceCondition, conditionType apiregistration.
	APIServiceConditionType) *apiregistration.APIServiceCondition {
	for i, condition := range conditions {
		if condition.Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}

// StatusREST implements the REST endpoint for changing the status of an APIService.
type StatusREST struct {
	store *genericregistry.Store
}

// NewStatusREST makes a RESTStorage for status that has more limited options.
// It is based on the original REST so that we can share the same underlying store.
func NewStatusREST(scheme *runtime.Scheme, rest *REST) *StatusREST {
	strategy := apiservice.NewStatusStrategy(scheme)
	statusStore := *rest.Store
	statusStore.CreateStrategy = nil
	statusStore.DeleteStrategy = nil
	statusStore.UpdateStrategy = strategy
	statusStore.ResetFieldsStrategy = strategy
	return &StatusREST{&statusStore}
}

var _ rest.Patcher = &StatusREST{} // New(), Get(), Update()

var _ rest.Storage = &StatusREST{} // New(), Destroy()
// New creates a new APIService object.
func (statusREST *StatusREST) New() runtime.Object {
	return &apiregistration.APIService{}
}

// Get retrieves the object from the storage. It is required to support Patch.
func (statusREST *StatusREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return statusREST.store.Get(ctx, name, options)
}

// Update alters the status subset of an object.
func (statusREST *StatusREST) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, _ bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	// We are explicitly setting forceAllowCreate to false in the call to the underlying storage because sub-resources
	// should never allow create on update
	return statusREST.store.Update(ctx, name, objInfo, createValidation, updateValidation, false, options)
}

// Destroy cleans up resources on shutdown.
func (statusREST *StatusREST) Destroy() {
}

var _ rest.ResetFieldsStrategy = &StatusREST{}

// GetResetFields implements rest.ResetFieldsStrategy
func (statusREST *StatusREST) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return statusREST.store.GetResetFields()
}
