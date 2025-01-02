package aggregator

import (
	"crypto/sha512"
	"fmt"
	"k8s.io/apiserver/pkg/authentication/user"
	endpointsRequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/util/responsewriter"
	"k8s.io/kube-openapi/pkg/cached"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"net/http"
	"strings"
	"sync/atomic"
)

type downloaderImplementation interface {
	handlerWithUser(httpHandler http.Handler, userinfo user.Info) http.Handler
	// Download downloads OpenAPI spec from /openapi/v2 endpoint of the given handler.
	// httpStatus is only valid if err == nil
	Download(httpHandler http.Handler, etag string) (returnSpec *spec.Swagger, newETag string, httpStatus int, err error)
}

var _ downloaderImplementation = &Downloader{}

// Downloader is the OpenAPI downloader type. It will try to download spec from /openapi/v2 or /swagger.json endpoint.
type Downloader struct {
}

// NewDownloader creates a new OpenAPI Downloader.
func NewDownloader() Downloader {
	return Downloader{}
}

func (downloader *Downloader) handlerWithUser(httpHandler http.Handler, userinfo user.Info) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		request = request.WithContext(endpointsRequest.WithUser(request.Context(), userinfo))
		httpHandler.ServeHTTP(response, request)
	})
}

// Download downloads OpenAPI spec from /openapi/v2 endpoint of the given handler.
// httpStatus is only valid if err == nil
func (downloader *Downloader) Download(httpHandler http.Handler, eTag string) (returnSpec *spec.Swagger, newETag string,
	httpStatus int, err error) {
	httpHandler = downloader.handlerWithUser(httpHandler, &user.DefaultInfo{Name: aggregatorUser})
	httpHandler = http.TimeoutHandler(httpHandler, specDownloadTimeout, "request timed out")
	request, err := http.NewRequest("GET", "/openapi/v2", nil)
	if err != nil {
		return nil, "", 0, err
	}
	request.Header.Add("Accept", "application/json")

	// Only pass eTag if it is not generated locally
	if len(eTag) > 0 && !strings.HasPrefix(eTag, locallyGeneratedEtagPrefix) {
		request.Header.Add("If-None-Match", eTag)
	}

	response := responsewriter.NewInMemoryResponseWriter()
	httpHandler.ServeHTTP(response, request)

	switch response.RespCode() {
	case http.StatusNotModified:
		if len(eTag) == 0 {
			return nil, eTag, http.StatusNotModified, fmt.Errorf("http.StatusNotModified is not " +
				"allowed in absence of eTag")
		}
		return nil, eTag, http.StatusNotModified, nil
	case http.StatusNotFound:
		// Gracefully skip 404, assuming the server won't provide any spec
		return nil, "", http.StatusNotFound, nil
	case http.StatusOK:
		openAPISpec := &spec.Swagger{}
		if err := openAPISpec.UnmarshalJSON(response.Data()); err != nil {
			return nil, "", 0, err
		}
		newETag := response.Header().Get("Etag")
		if len(newETag) == 0 {
			newETag = eTagFor(response.Data())
			if len(eTag) > 0 && strings.HasPrefix(eTag, locallyGeneratedEtagPrefix) {
				if eTag == newETag {
					// The function call with an eTag and server does not report an eTag.
					// That means this server does not support eTag and the eTag that passed to the function generated
					// previously by us. Just compare eTags and return StatusNotModified if they are the same.
					return nil, eTag, http.StatusNotModified, nil
				}
			}
		}
		return openAPISpec, newETag, http.StatusOK, nil
	default:
		return nil, "", 0, fmt.Errorf("failed to retrieve openAPI spec, "+
			"http error: %s", response.String())
	}
}

func eTagFor(data []byte) string {
	return fmt.Sprintf("%s%X\"", locallyGeneratedEtagPrefix, sha512.Sum512(data))
}

type CacheableDownloader interface {
	UpdateHandler(http.Handler)
	cached.Value[*spec.Swagger]
}

func NewCacheableDownloader(apiServiceName string, downloader *Downloader, handler http.Handler) CacheableDownloader {
	c := &cacheableDownloader{
		name:       apiServiceName,
		downloader: downloader,
	}
	c.handler.Store(&handler)
	return c
}

// cacheableDownloader implements CacheableDownloader interface.
var _ CacheableDownloader = &cacheableDownloader{}

// cacheableDownloader is downloader that will always return the data and eTag.
type cacheableDownloader struct {
	name       string
	downloader *Downloader
	// handler is the http.Handler for the API service that can be replaced
	handler atomic.Pointer[http.Handler]
	eTag    string
	spec    *spec.Swagger
}

func (cacheable *cacheableDownloader) UpdateHandler(handler http.Handler) {
	cacheable.handler.Store(&handler)
}
func (cacheable *cacheableDownloader) Get() (*spec.Swagger, string, error) {
	if swaggerSpec, eTag, err := cacheable.get(); err != nil {
		return swaggerSpec, eTag, fmt.Errorf("failed to download %v: %v", cacheable.name, err)
	} else {
		return swaggerSpec, eTag, err
	}
}

func (cacheable *cacheableDownloader) get() (*spec.Swagger, string, error) {
	handler := *cacheable.handler.Load()
	swagger, eTag, status, err := cacheable.downloader.Download(handler, cacheable.eTag)
	if err != nil {
		return nil, "", err
	}
	switch status {
	case http.StatusNotModified:
	// Nothing has changed, do nothing.
	case http.StatusOK:
		if swagger != nil {
			cacheable.eTag = eTag
			cacheable.spec = swagger
			break
		}
		fallthrough
	case http.StatusNotFound:
		return nil, "", ErrAPIServiceNotFound
	default:
		return nil, "", fmt.Errorf("invalid status code: %v", status)
	}
	return cacheable.spec, cacheable.eTag, nil
}
