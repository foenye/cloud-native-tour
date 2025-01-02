package aggregator

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"k8s.io/kube-openapi/pkg/handler3"
	"net/http"
	"testing"
)

var _ http.Handler = handlerTest{}

type handlerTest struct {
	eTag string
	data []byte
}

func (h handlerTest) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	// Create an APIService with a handler for one group/version
	if request.URL.Path == "/openapi/v3" {
		group := &handler3.OpenAPIV3Discovery{
			Paths: map[string]handler3.OpenAPIV3DiscoveryGroupVersion{
				"apis/group/version": {
					ServerRelativeURL: "/openapi/v3/apis/group/version?hash=" + h.eTag,
				},
			},
		}
		groupBytes, _ := json.Marshal(group)
		_, _ = response.Write(groupBytes)
		return
	}
	if request.URL.Path == "/openapi/v3/apis/group/version" {
		if len(h.eTag) > 0 {
			response.Header().Add("Etag", h.eTag)
		}
		ifNoneMatches := request.Header["If-None-Match"]
		for _, match := range ifNoneMatches {
			if match == h.eTag {
				response.WriteHeader(http.StatusNotModified)
				return
			}
		}
		_, _ = response.Write(h.data)
	}
}

func TestDownloader_OpenAPIV3Root(t *testing.T) {
	downloader := NewDownloader()
	groups, _, err := downloader.OpenAPIV3Root(handlerTest{data: []byte(""), eTag: ""})
	assert.NoError(t, err)
	if assert.NotNil(t, groups) {
		assert.Len(t, groups.Paths, 1)
		if assert.Contains(t, groups.Paths, "apis/group/version") {
			assert.NotEmpty(t, groups.Paths["apis/group/version"].ServerRelativeURL)
		}
	}
}
