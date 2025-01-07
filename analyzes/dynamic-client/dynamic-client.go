package main

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
	panicErr(err)

	dynamicClient, err := dynamic.NewForConfig(config)
	panicErr(err)

	gvr := schema.GroupVersionResource{Version: "v1", Resource: "pods"}
	unstructuredList, err := dynamicClient.Resource(gvr).Namespace(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{Limit: 500})
	panicErr(err)

	pods := &corev1.PodList{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredList.UnstructuredContent(), pods)
	panicErr(err)

	fmt.Println("NAMESPACE, \t\t\t\t NAME \t\t\t STATUS")
	for _, pod := range pods.Items {
		fmt.Printf("%v \t\t\t %v \t\t\t %+v\n", pod.Namespace, pod.Name, pod.Status.Phase)
	}
}
