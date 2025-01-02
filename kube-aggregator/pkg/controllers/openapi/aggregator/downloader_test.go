package aggregator

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"net/http"
	"testing"
)

var _ http.Handler = handlerTest{}

type handlerTest struct {
	eTag string
	data []byte
}

func (handler handlerTest) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if len(handler.eTag) > 0 {
		response.Header().Add("Etag", handler.eTag)
	}
	ifNonMatches := request.Header["If-None-Match"]
	for _, match := range ifNonMatches {
		if match == handler.eTag {
			response.WriteHeader(http.StatusNotModified)
			return
		}
	}
	_, _ = response.Write(handler.data)
}

var _ http.Handler = handlerDeprecatedTest{}

type handlerDeprecatedTest struct {
	eTag string
	data []byte
}

func (handler handlerDeprecatedTest) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	// old server returns 403 on new endpoint
	if request.URL.Path == "/openapi/v2" {
		response.WriteHeader(http.StatusForbidden)
		return
	}
	if len(handler.eTag) > 0 {
		response.Header().Add("Etag", handler.eTag)
	}
	ifNonMatches := request.Header["If-None-Match"]
	for _, match := range ifNonMatches {
		if match == handler.eTag {
			response.WriteHeader(http.StatusNotModified)
			return
		}
	}
	_, _ = response.Write(handler.data)
}

func assertDownloadSpec(actualSpec *spec.Swagger, actualETag string, err error, expectedSpecID string,
	expectedETag string) error {
	if err != nil {
		return fmt.Errorf("downloadOpenAPISpec failed: %s", err)
	}
	if expectedSpecID == "" && actualSpec != nil {
		return fmt.Errorf("expected Not Modified, actual ID %s", actualSpec.ID)
	}
	if actualSpec != nil && actualSpec.ID != expectedSpecID {
		return fmt.Errorf("expected ID %s, actual ID %s", expectedSpecID, actualSpec.ID)
	}
	if actualETag != expectedETag {
		return fmt.Errorf("expected ETag `%s`, actual ETag `%s`", expectedETag, actualETag)
	}
	return nil
}

func TestDownloadOpenAPISpec(t *testing.T) {
	downloader := Downloader{}

	// Test with no eTag
	actualSpec, actualETag, _, err := downloader.Download(handlerTest{data: []byte(`{"id": "test"}`)}, "")
	assert.NoError(t, assertDownloadSpec(actualSpec, actualETag, err, "test", `"6E8F849B434D4B98A569B9D7718876E9-356ECAB19D7FBE1336BABB1E70F8F3025050DE218BE78256BE81620681CFC9A268508E542B8B55974E17B2184BBFC8FFFAA577E51BE195D32B3CA2547818ABE4"`))

	// Test with eTag
	actualSpec, actualETag, _, err = downloader.Download(handlerTest{data: []byte(`{"id": "test"}`), eTag: "etag_test"}, "")
	assert.NoError(t, assertDownloadSpec(actualSpec, actualETag, err, "test", `etag_test`))

	// Test with not modified
	actualSpec, actualETag, _, err = downloader.Download(handlerTest{data: []byte(`{"id": "test"}`), eTag: "etag_test"}, "etag_test")
	assert.NoError(t, assertDownloadSpec(actualSpec, actualETag, err, "", `etag_test`))

	// Test with different eTags
	actualSpec, actualETag, _, err = downloader.Download(handlerTest{data: []byte(`{"id": "test"}`), eTag: "etag_test1"}, "etag_test2")
	assert.NoError(t, assertDownloadSpec(actualSpec, actualETag, err, "test", `etag_test1`))
}
