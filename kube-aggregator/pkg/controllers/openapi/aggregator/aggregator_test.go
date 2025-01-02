package aggregator

import (
	"bytes"
	"encoding/json"
	apiregistrationv1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1"
	"io"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const (
	dummy                      = "dummy"
	apiServiceName             = "apiservice"
	openapiV2                  = "/openapi/v2"
	apisFooV1                  = "/apis/foo/v1"
	apisAPIRegistrationK8sIoV1 = "/apis/apiregistration.k8s.io/v1/"
	apisAPIServiceGroupV1Path1 = "/apis/apiservicegroup/v1/path1"
	apisAPIServiceGroupV1Path2 = "/apis/apiservicegroup/v1/path2"
)

var (
	onlyContainsAPIsFooV1DelegationHandler = &openAPIHandler{
		openapi: &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Paths: &spec.Paths{
					Paths: map[string]spec.PathItem{
						apisFooV1: {},
					},
				},
			},
		},
	}
)

var _ http.Handler = &openAPIHandler{}

type openAPIHandler struct {
	delaySeconds int
	openapi      *spec.Swagger
	returnErr    bool
}

func (o *openAPIHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	time.Sleep(time.Duration(o.delaySeconds) * time.Second)
	if o.returnErr {
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(o.openapi)
	if err != nil {
		panic(err)
	}
	http.ServeContent(response, request, openapiV2, time.Now(), bytes.NewReader(data))
	return
}

func TestBasicPathsMerged(t *testing.T) {
	mux := http.NewServeMux()
	delegationHandlers := []http.Handler{onlyContainsAPIsFooV1DelegationHandler}
	buildAndRegisterSpecAggregator(delegationHandlers, mux)

	swagger, err := fetchOpenAPI(mux)
	if err != nil {
		t.Error(err)
	}
	expectPath(t, swagger, apisFooV1)
	expectPath(t, swagger, apisAPIRegistrationK8sIoV1)
}

func TestSpecAggregator_AddUpdateAPIService(t *testing.T) {
	mux := http.NewServeMux()
	var delegationHandlers []http.Handler
	delegationHandlers = append(delegationHandlers, onlyContainsAPIsFooV1DelegationHandler)
	aggregator := buildAndRegisterSpecAggregator(delegationHandlers, mux)

	apiService := &apiregistrationv1.APIService{
		Spec: apiregistrationv1.APIServiceSpec{
			Group:   "apiservicegroup",
			Version: "v1",
			Service: &apiregistrationv1.ServiceReference{Name: dummy},
		},
	}
	apiService.Name = apiServiceName

	delegationHandler := &openAPIHandler{openapi: &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Paths: &spec.Paths{
				Paths: map[string]spec.PathItem{
					apisAPIServiceGroupV1Path1: {},
				},
			},
		},
	}}

	if err := aggregator.AddUpdateAPIService(apiService, delegationHandler); err != nil {
		t.Error(err)
	}
	if err := aggregator.UpdateAPIServiceSpec(apiService.Name); err != nil {
		t.Error(err)
	}

	swagger, err := fetchOpenAPI(mux)
	if err != nil {
		t.Error(err)
	}

	expectPath(t, swagger, apisAPIServiceGroupV1Path1)
	expectPath(t, swagger, apisAPIRegistrationK8sIoV1)

	t.Log("Update APIService OpenAPI")
	delegationHandler.openapi = &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Paths: &spec.Paths{
				Paths: map[string]spec.PathItem{
					apisAPIServiceGroupV1Path2: {},
				},
			},
		},
	}
	if err := aggregator.UpdateAPIServiceSpec(apiService.Name); err != nil {
		t.Error(err)
	}

	swagger, err = fetchOpenAPI(mux)
	if err != nil {
		t.Error(err)
	}
	// Ensure that the if the APIService OpenAPI is updated, the
	// aggregated OpenAPI is also updated.
	expectPath(t, swagger, apisAPIServiceGroupV1Path2)
	expectNoPath(t, swagger, apisAPIServiceGroupV1Path1)
	expectPath(t, swagger, apisAPIRegistrationK8sIoV1)
}

// Tests that an APIService that registers OpenAPI will only have the OpenAPI for its specific group version
// served registered.
// See https://github.com/kubernetes/kubernetes/pull/123570 for full context.
func TestAPIServiceOpenAPIServiceMismatch(t *testing.T) {
	mux := http.NewServeMux()
	var delegationHandlers []http.Handler
	delegationHandlers = append(delegationHandlers, onlyContainsAPIsFooV1DelegationHandler)

	aggregator := buildAndRegisterSpecAggregator(delegationHandlers, mux)

	apiService := &apiregistrationv1.APIService{
		Spec: apiregistrationv1.APIServiceSpec{
			Group:   "apiservicegroup",
			Version: "v1",
			Service: &apiregistrationv1.ServiceReference{Name: "dummy"},
		},
	}
	apiService.Name = "apiservice"

	apiService2 := &apiregistrationv1.APIService{
		Spec: apiregistrationv1.APIServiceSpec{
			Group:   "apiservicegroup",
			Version: "v2",
			Service: &apiregistrationv1.ServiceReference{Name: "dummy2"},
		},
	}
	apiService2.Name = "apiservice2"

	handler := &openAPIHandler{openapi: &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Paths: &spec.Paths{
				Paths: map[string]spec.PathItem{
					"/apis/apiservicegroup/v1/":      {},
					"/apis/apiservicegroup/v1beta1/": {},
				},
			},
		},
	}}

	handler2 := &openAPIHandler{openapi: &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Paths: &spec.Paths{
				Paths: map[string]spec.PathItem{
					"/apis/a/":                  {},
					"/apis/apiservicegroup/v1/": {},
					"/apis/apiservicegroup/v2/": {},
				},
			},
		},
	}}

	if err := aggregator.AddUpdateAPIService(apiService, handler); err != nil {
		t.Error(err)
	}
	if err := aggregator.UpdateAPIServiceSpec(apiService.Name); err != nil {
		t.Error(err)
	}

	if err := aggregator.AddUpdateAPIService(apiService2, handler2); err != nil {
		t.Error(err)
	}
	if err := aggregator.UpdateAPIServiceSpec(apiService2.Name); err != nil {
		t.Error(err)
	}

	swagger, err := fetchOpenAPI(mux)
	if err != nil {
		t.Error(err)
	}
	expectPath(t, swagger, "/apis/apiservicegroup/v1/")
	expectPath(t, swagger, "/apis/apiservicegroup/v2/")
	expectPath(t, swagger, "/apis/apiregistration.k8s.io/v1/")
	expectNoPath(t, swagger, "/apis/a/")
	expectNoPath(t, swagger, "/apis/apiservicegroup/v1beta1/")

	t.Logf("Remove APIService %s", apiService.Name)
	aggregator.RemoveAPIService(apiService.Name)

	swagger, err = fetchOpenAPI(mux)
	if err != nil {
		t.Error(err)
	}
	// Ensure that the if the APIService is added then removed, the OpenAPI disappears from the aggregated OpenAPI as well.
	expectNoPath(t, swagger, "/apis/apiservicegroup/v1")
	expectPath(t, swagger, "/apis/apiregistration.k8s.io/v1/")
	expectNoPath(t, swagger, "/apis/a")
}

func TestAddRemoveAPIService(t *testing.T) {
	mux := http.NewServeMux()
	var delegationHandlers []http.Handler
	delegationHandlers = append(delegationHandlers, onlyContainsAPIsFooV1DelegationHandler)

	s := buildAndRegisterSpecAggregator(delegationHandlers, mux)

	apiService := &apiregistrationv1.APIService{
		Spec: apiregistrationv1.APIServiceSpec{
			Group:   "apiservicegroup",
			Version: "v1",
			Service: &apiregistrationv1.ServiceReference{Name: "dummy"},
		},
	}
	apiService.Name = "apiservice"

	handler := &openAPIHandler{openapi: &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Paths: &spec.Paths{
				Paths: map[string]spec.PathItem{
					"/apis/apiservicegroup/v1/": {},
				},
			},
		},
	}}

	if err := s.AddUpdateAPIService(apiService, handler); err != nil {
		t.Error(err)
	}
	if err := s.UpdateAPIServiceSpec(apiService.Name); err != nil {
		t.Error(err)
	}

	swagger, err := fetchOpenAPI(mux)
	if err != nil {
		t.Error(err)
	}
	expectPath(t, swagger, "/apis/apiservicegroup/v1/")
	expectPath(t, swagger, "/apis/apiregistration.k8s.io/v1/")

	t.Logf("Remove APIService %s", apiService.Name)
	s.RemoveAPIService(apiService.Name)

	swagger, err = fetchOpenAPI(mux)
	if err != nil {
		t.Error(err)
	}
	// Ensure that the if the APIService is added then removed, the OpenAPI disappears from the aggregated OpenAPI as well.
	expectNoPath(t, swagger, "/apis/apiservicegroup/v1/")
	expectPath(t, swagger, "/apis/apiregistration.k8s.io/v1/")
}

func TestUpdateAPIService(t *testing.T) {
	mux := http.NewServeMux()
	var delegationHandlers []http.Handler
	delegationHandlers = append(delegationHandlers, onlyContainsAPIsFooV1DelegationHandler)

	s := buildAndRegisterSpecAggregator(delegationHandlers, mux)

	apiService := &apiregistrationv1.APIService{
		Spec: apiregistrationv1.APIServiceSpec{
			Group:   "apiservicegroup",
			Version: "v1",
			Service: &apiregistrationv1.ServiceReference{Name: "dummy"},
		},
	}
	apiService.Name = "apiservice"

	handler := &openAPIHandler{openapi: &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Paths: &spec.Paths{
				Paths: map[string]spec.PathItem{
					"/apis/apiservicegroup/v1/": {},
				},
			},
		},
	}}

	handler2 := &openAPIHandler{openapi: &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Paths: &spec.Paths{
				Paths: map[string]spec.PathItem{},
			},
		},
	}}

	if err := s.AddUpdateAPIService(apiService, handler); err != nil {
		t.Error(err)
	}
	if err := s.UpdateAPIServiceSpec(apiService.Name); err != nil {
		t.Error(err)
	}

	swagger, err := fetchOpenAPI(mux)
	if err != nil {
		t.Error(err)
	}
	expectPath(t, swagger, "/apis/apiservicegroup/v1/")
	expectPath(t, swagger, "/apis/apiregistration.k8s.io/v1/")

	t.Logf("Updating APIService %s", apiService.Name)
	if err := s.AddUpdateAPIService(apiService, handler2); err != nil {
		t.Error(err)
	}
	if err := s.UpdateAPIServiceSpec(apiService.Name); err != nil {
		t.Error(err)
	}

	swagger, err = fetchOpenAPI(mux)
	if err != nil {
		t.Error(err)
	}
	// Ensure that the if the APIService is added and then handler is modified, the new data is reflected in the aggregated OpenAPI.
	expectNoPath(t, swagger, "/apis/apiservicegroup/v1/")
	expectPath(t, swagger, "/apis/apiregistration.k8s.io/v1/")
}

func TestFailingAPIServiceSkippedAggregation(t *testing.T) {
	mux := http.NewServeMux()
	var delegationHandlers []http.Handler
	delegationHandlers = append(delegationHandlers, onlyContainsAPIsFooV1DelegationHandler)

	s := buildAndRegisterSpecAggregator(delegationHandlers, mux)

	apiServiceFailed := &apiregistrationv1.APIService{
		Spec: apiregistrationv1.APIServiceSpec{
			Group:   "failed",
			Version: "v1",
			Service: &apiregistrationv1.ServiceReference{Name: "dummy"},
		},
	}
	apiServiceFailed.Name = "apiserviceFailed"

	handlerFailed := &openAPIHandler{
		returnErr: true,
		openapi: &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Paths: &spec.Paths{
					Paths: map[string]spec.PathItem{
						"/apis/failed/v1/": {},
					},
				},
			},
		},
	}

	apiServiceSuccess := &apiregistrationv1.APIService{
		Spec: apiregistrationv1.APIServiceSpec{
			Group:   "success",
			Version: "v1",
			Service: &apiregistrationv1.ServiceReference{Name: "dummy2"},
		},
	}
	apiServiceSuccess.Name = "apiserviceSuccess"

	handlerSuccess := &openAPIHandler{
		openapi: &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Paths: &spec.Paths{
					Paths: map[string]spec.PathItem{
						"/apis/success/v1/": {},
					},
				},
			},
		},
	}

	if err := s.AddUpdateAPIService(apiServiceSuccess, handlerSuccess); err != nil {
		t.Error(err)
	}
	if err := s.AddUpdateAPIService(apiServiceFailed, handlerFailed); err != nil {
		t.Error(err)
	}
	if err := s.UpdateAPIServiceSpec(apiServiceSuccess.Name); err != nil {
		t.Error(err)
	}
	err := s.UpdateAPIServiceSpec(apiServiceFailed.Name)
	if err == nil {
		t.Errorf("Expected updating failing apiService %s to return error", apiServiceFailed.Name)
	}

	swagger, err := fetchOpenAPI(mux)
	if err != nil {
		t.Error(err)
	}
	expectPath(t, swagger, "/apis/foo/v1/")
	expectNoPath(t, swagger, "/apis/failed/v1/")
	expectPath(t, swagger, "/apis/success/v1/")
}

func TestAPIServiceFailSuccessTransition(t *testing.T) {
	mux := http.NewServeMux()
	var delegationHandlers []http.Handler
	delegationHandlers = append(delegationHandlers, onlyContainsAPIsFooV1DelegationHandler)

	s := buildAndRegisterSpecAggregator(delegationHandlers, mux)

	apiService := &apiregistrationv1.APIService{
		Spec: apiregistrationv1.APIServiceSpec{
			Group:   "apiservicegroup",
			Version: "v1",
			Service: &apiregistrationv1.ServiceReference{Name: "dummy"},
		},
	}
	apiService.Name = "apiservice"

	handler := &openAPIHandler{
		returnErr: true,
		openapi: &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Paths: &spec.Paths{
					Paths: map[string]spec.PathItem{
						"/apis/apiservicegroup/v1/": {},
					},
				},
			},
		},
	}

	if err := s.AddUpdateAPIService(apiService, handler); err != nil {
		t.Error(err)
	}
	if err := s.UpdateAPIServiceSpec(apiService.Name); err == nil {
		t.Errorf("Expected error for when updating spec for failing apiservice")
	}

	swagger, err := fetchOpenAPI(mux)
	if err != nil {
		t.Error(err)
	}
	expectPath(t, swagger, "/apis/foo/v1/")
	expectNoPath(t, swagger, "/apis/apiservicegroup/v1/")

	t.Log("Transition APIService to not return error")
	handler.returnErr = false
	err = s.UpdateAPIServiceSpec(apiService.Name)
	if err != nil {
		t.Error(err)
	}
	swagger, err = fetchOpenAPI(mux)
	if err != nil {
		t.Error(err)
	}
	expectPath(t, swagger, "/apis/foo/v1/")
	expectPath(t, swagger, "/apis/apiservicegroup/v1/")
}

func TestFailingAPIServiceDoesNotBlockAdd(t *testing.T) {
	mux := http.NewServeMux()
	var delegationHandlers []http.Handler
	delegate1 := &openAPIHandler{openapi: &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Paths: &spec.Paths{
				Paths: map[string]spec.PathItem{
					"/apis/foo/v1/": {},
				},
			},
		},
	}}
	delegationHandlers = append(delegationHandlers, delegate1)

	s := buildAndRegisterSpecAggregator(delegationHandlers, mux)

	apiServiceFailed := &apiregistrationv1.APIService{
		Spec: apiregistrationv1.APIServiceSpec{
			Group:   "failed",
			Version: "v1",
			Service: &apiregistrationv1.ServiceReference{Name: "dummy"},
		},
	}
	apiServiceFailed.Name = "apiserviceFailed"

	// Create a handler that has a long response time and ensure that
	// adding the APIService does not block.
	handlerFailed := &openAPIHandler{
		delaySeconds: 5,
		returnErr:    true,
		openapi: &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Paths: &spec.Paths{
					Paths: map[string]spec.PathItem{
						"/apis/failed/v1/": {},
					},
				},
			},
		},
	}

	updateDone := make(chan bool)
	go func() {
		if err := s.AddUpdateAPIService(apiServiceFailed, handlerFailed); err != nil {
			t.Error(err)
		}
		close(updateDone)
	}()

	select {
	case <-updateDone:
	case <-time.After(2 * time.Second):
		t.Errorf("AddUpdateAPIService affected by APIService response time")
	}

	swagger, err := fetchOpenAPI(mux)
	if err != nil {
		t.Error(err)
	}
	expectPath(t, swagger, "/apis/foo/v1/")
	expectNoPath(t, swagger, "/apis/failed/v1/")
}

func expectNoPath(t *testing.T, swagger *spec.Swagger, path string) {
	if _, ok := swagger.Paths.Paths[path]; ok {
		t.Errorf("Expected path %s to be omitted in aggregated paths", path)
	}
}

func expectPath(t *testing.T, swagger *spec.Swagger, path string) {
	if _, exists := swagger.Paths.Paths[path]; !exists {
		t.Errorf("Expected paht %s to exist in aggregated paths", path)
	}
}

func fetchOpenAPI(serveMux *http.ServeMux) (*spec.Swagger, error) {
	server := httptest.NewServer(serveMux)
	defer server.Close()
	client := server.Client()

	request, err := http.NewRequest(http.MethodGet, server.URL+openapiV2, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(response.Body)

	swagger := &spec.Swagger{}
	if err := swagger.UnmarshalJSON(body); err != nil {
		return nil, err
	}
	return swagger, err
}

func buildAndRegisterSpecAggregator(delegationHandlers []http.Handler, serveMux *http.ServeMux) *specAggregator {
	downloader := NewDownloader()
	aggregatorSpec := &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Paths: &spec.Paths{
				Paths: map[string]spec.PathItem{
					apisAPIRegistrationK8sIoV1: {},
				},
			},
		},
	}
	return buildAndRegisterSpecAggregatorForLocalService(&downloader, aggregatorSpec, delegationHandlers, serveMux)
}
