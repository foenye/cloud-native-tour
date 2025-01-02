package aggregator

import (
	"encoding/json"
	"fmt"
	"k8s.io/apiserver/pkg/authentication/user"
	endpointsRequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/util/responsewriter"
	"k8s.io/kube-openapi/pkg/handler3"
	"net/http"
)

var _ error = &NotFoundError{}

type NotFoundError struct {
}

func (e *NotFoundError) Error() string {
	return ""
}

type downloaderImplementation interface {
	handlerWithUser(httpHandler http.Handler, userinfo user.Info) http.Handler
	OpenAPIV3Root(httpHandler http.Handler) (*handler3.OpenAPIV3Discovery, int, error)
}

var _ downloaderImplementation = &Downloader{}

// Downloader is the OpenAPI downloader type. It will try to download spec from /openapi/v3
// and /openapi/v3/<group>/<version> endpoints.
type Downloader struct {
}

// NewDownloader creates a new OpenAPI Downloader.
func NewDownloader() Downloader {
	return Downloader{}
}

func (loader *Downloader) handlerWithUser(httpHandler http.Handler, userinfo user.Info) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		request = request.WithContext(endpointsRequest.WithUser(request.Context(), userinfo))
		httpHandler.ServeHTTP(response, request)
	})
}

// OpenAPIV3Root downloads the OpenAPI V3 root document from an APIService.
func (loader *Downloader) OpenAPIV3Root(httpHandler http.Handler) (*handler3.OpenAPIV3Discovery, int, error) {
	httpHandler = loader.handlerWithUser(httpHandler, &user.DefaultInfo{Name: aggregatorUser})
	httpHandler = http.TimeoutHandler(httpHandler, specDownloadTimeout, "request timed out")

	request, err := http.NewRequest(http.MethodGet, "/openapi/v3", nil)
	if err != nil {
		return nil, 0, err
	}
	response := responsewriter.NewInMemoryResponseWriter()
	httpHandler.ServeHTTP(response, request)

	switch response.RespCode() {
	case http.StatusNotFound:
		return nil, response.RespCode(), nil
	case http.StatusOK:
		groups := handler3.OpenAPIV3Discovery{}
		if err := json.Unmarshal(response.Data(), &groups); err != nil {
			return nil, response.RespCode(), err
		}
		return &groups, response.RespCode(), nil
	}
	return nil, response.RespCode(), fmt.Errorf("%s, could not get list of group versions for APIService",
		"Error")
}
