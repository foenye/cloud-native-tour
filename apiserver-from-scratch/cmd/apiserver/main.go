package main

import (
	"github.com/yeahfo/cloud-native-tour/apiserver-from-scratch/cmd/apis"
	"github.com/yeahfo/cloud-native-tour/apiserver-from-scratch/cmd/foos"
	"github.com/yeahfo/cloud-native-tour/apiserver-from-scratch/pkg/helper"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const (
	tlsKeyName  = "apiserver.key"
	tlsCertName = "apiserver.crt"
)

// @title           hello.yeahfo.github.io-server
// @version         0.1
// @description     K8s apiserver style http server from scratch
// @BasePath  /apis
func main() {
	mux := ServeMux()
	if certDir := os.Getenv("CERT_DIR"); certDir != "" {
		tlsCertFile := filepath.Join(certDir, tlsCertName)
		tlsKeyFile := filepath.Join(certDir, tlsKeyName)
		log.Println("serving https on 0.0.0.0:6443")
		log.Fatal(http.ListenAndServeTLS(":6443", tlsCertFile, tlsKeyFile, mux))
	} else {
		log.Println("serving http on 0.0.0.0:8000")
		log.Fatal(http.ListenAndServe(":8000", mux))
	}
}

func ServeMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/", helper.LoggerHandler(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "" && request.URL.Path != "/" {
			responseWriter.WriteHeader(http.StatusNotFound)
			return
		}
	})))

	mux.Handle("/apis", helper.LoggerHandler(http.HandlerFunc(apis.APIs)))
	mux.Handle("/apis/hello.yeahfo.github.io", helper.LoggerHandler(http.HandlerFunc(apis.APIGroup)))
	mux.Handle("/apis/hello.yeahfo.github.io/v1", helper.LoggerHandler(http.HandlerFunc(apis.APIGroupV1Resources)))
	mux.Handle("/openapi/v2", helper.LoggerHandler(http.HandlerFunc(apis.OpenapiV2)))

	// LIST /apis/hello.yeahfo.github.io/v1/foos
	// LIST /apis/hello.yeahfo.github.io/v1/namespaces/{namespace}/foos
	// GET  /apis/hello.yeahfo.github.io/v1/namespaces/{namespace}/foos/{name}
	// POST /apis/hello.yeahfo.github.io/v1/namespaces/{namespace}/foos/
	// PUT  /apis/hello.yeahfo.github.io/v1/namespaces/{namespace}/foos/{name}
	// DEL  /apis/hello.yeahfo.github.io/v1/namespaces/{namespace}/foos/{name}
	mux.Handle("/apis/hello.yeahfo.github.io/v1/", helper.LoggerHandler(helper.ContentTypeJSONHandler(http.HandlerFunc(
		foos.FooHandlers))))

	return mux
}
