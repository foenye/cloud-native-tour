package helper

import (
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	openapiv2 "github.com/google/gnostic-models/openapiv2"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
)

func RenderJSON(responseWriter http.ResponseWriter, value interface{}) {
	jsonEncodedBytes, err := json.Marshal(value)
	if err != nil {
		WriteErrorStatus(responseWriter, "", http.StatusInternalServerError, err.Error())
		return
	}
	_, _ = responseWriter.Write(jsonEncodedBytes)
}

//var status = metav1.Status{
//	TypeMeta: metav1.TypeMeta{
//		APIVersion: "v1",
//		Kind:       "Status",
//	},
//	Status: metav1.StatusFailure,
//	Details: &metav1.StatusDetails{
//		Group: "hello.yeahfo.github.io",
//		Kind:  "foos",
//	},
//}

const errorStatusTemplate = `
{
	"apiVersion": "v1",
	"kind": "Status",
	"metadata": {},
	"status": "Failure",
	"message": "%s",
	"reason": "%s",
	"details": {"group": "hello.yeahfo.github.io", "kind": "foos", "name": "%s"},
	"code": %d
}
`

func WriteErrorStatus(responseWriter http.ResponseWriter, name string, httpErrorCode int, errorMessage string) {
	var errorStatus string
	switch httpErrorCode {
	case http.StatusNotFound:
		errorStatus = fmt.Sprintf(errorStatusTemplate, fmt.Sprintf(`foos '%s' not found`, name),
			http.StatusText(http.StatusNotFound), name, http.StatusNotFound)
	case http.StatusConflict:
		errorStatus = fmt.Sprintf(errorStatusTemplate, fmt.Sprintf(`foos '%s' already exists`, name),
			http.StatusText(http.StatusConflict), name, http.StatusConflict)
	default:
		errorStatus = fmt.Sprintf(errorStatusTemplate, errorMessage, http.StatusText(httpErrorCode), name, httpErrorCode)
	}
	responseWriter.WriteHeader(httpErrorCode)
	_, _ = responseWriter.Write([]byte(errorStatus))
}

func ToProtoBinary(encodedJsonBytes []byte) ([]byte, error) {
	document, err := openapiv2.ParseDocument(encodedJsonBytes)
	if err != nil {
		return nil, err
	}
	return proto.Marshal(document)
}

func LoggerHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		dumpedRequest, _ := httputil.DumpRequest(request, true)
		dumped := string(dumpedRequest)
		if idx := strings.Index(dumped, "\n"); idx != -1 {
			log.Println("rx", dumped[:idx])
			log.Println("rx Content-type:", request.Header.Get("Content-Type"))
			log.Println("rx Accept:", request.Header.Get("Accept"))
			for name, values := range request.Header {
				if strings.Index(name, "X-") == 0 {
					log.Printf("rx %s:%v", name, values)
				}
			}
			log.Println()
		} else {
			log.Println("rx", dumped)
		}
		handler.ServeHTTP(responseWriter, request)
	})
}

func ContentTypeJSONHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("Content-Type", "application/json")
		handler.ServeHTTP(responseWriter, request)
	})
}
