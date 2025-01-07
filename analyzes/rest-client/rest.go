package main

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

func PanicErr(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {

	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
	PanicErr(err)
	config.APIPath = "api"
	config.GroupVersion = &corev1.SchemeGroupVersion
	config.NegotiatedSerializer = scheme.Codecs

	restClient, err := rest.RESTClientFor(config)
	PanicErr(err)

	pods := &corev1.PodList{}

	err = restClient.Get().Namespace(metav1.NamespaceAll).Resource(corev1.ResourcePods.String()).VersionedParams(&metav1.
		ListOptions{Limit: 500}, scheme.ParameterCodec).Do(context.TODO()).Into(pods)
	PanicErr(err)

	fmt.Println("NAMESPACE, \t\t\t\t NAME \t\t\t STATUS")
	for _, pod := range pods.Items {
		fmt.Printf("%v, \t\t\t %v \t\t\t %+v\n", pod.Namespace, pod.Name, pod.Status.Phase)
	}
}
