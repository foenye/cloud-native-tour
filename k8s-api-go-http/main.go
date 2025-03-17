package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	serializerjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/klog/v2"
	"net/http"
)

func main() {
	if err := createPod(); err != nil {
		klog.Error(err)
	}
}

func createPod() error {
	pod := createPodObject()
	serializer := getJSONSerializer()
	createPodCommand, err := serializePodObject(serializer, pod)
	if err != nil {
		return err
	}
	createRequest, err := buildCreateRequest(createPodCommand)
	if err != nil {
		return err
	}

	client := &http.Client{}
	response, err := client.Do(createRequest)
	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)

	created, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode < 300 {
		represent, err := deserializePod(serializer, created)
		if err != nil {
			return err
		}

		jsonPod, err := json.MarshalIndent(represent, "", "  ")
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", jsonPod)
	} else {
		status, err := deserializeStatusBody(serializer, created)
		if err != nil {
			return err
		}
		jsonStatus, err := json.MarshalIndent(status, "", " ")
		fmt.Printf("%s\n", jsonStatus)
	}
	return nil

}

func deserializeStatusBody(serializer runtime.Serializer, created []byte) (*metav1.Status, error) {
	var status metav1.Status
	_, _, err := serializer.Decode(created, nil, &status)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

func deserializePod(serializer runtime.Serializer, created []byte) (*corev1.Pod, error) {
	var pod corev1.Pod
	_, _, err := serializer.Decode(created, nil, &pod)
	if err != nil {
		return nil, err
	}
	return &pod, nil
}

func buildCreateRequest(createPodCommand io.Reader) (*http.Request, error) {
	request, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:8001/api/v1/namespaces/default/pods", createPodCommand)
	if err != nil {
		return nil, err
	}
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")
	return request, nil
}

func serializePodObject(serializer runtime.Serializer, pod *corev1.Pod) (io.Reader, error) {
	var buf bytes.Buffer
	if err := serializer.Encode(pod, &buf); err != nil {
		return nil, err
	}
	return &buf, nil
}

func getJSONSerializer() runtime.Serializer {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(
		schema.GroupVersion{Group: "", Version: "v1"},
		&corev1.Pod{},
		&metav1.Status{},
	)
	return serializerjson.NewSerializerWithOptions(serializerjson.SimpleMetaFactory{}, nil, scheme, serializerjson.SerializerOptions{})
}
func createPodObject() *corev1.Pod {
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "runtime",
					Image: "nginx:stable",
				},
			},
		},
	}
	pod.SetName("my-pod")
	pod.SetLabels(map[string]string{
		"app.kubernetes.io/component": "my-component",
		"app.kubernetes.io/name":      "a-name",
	})
	return &pod
}
