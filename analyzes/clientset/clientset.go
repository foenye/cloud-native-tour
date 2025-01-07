package main

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

	clientset, err := kubernetes.NewForConfig(config)
	panicErr(err)
	podResource := clientset.CoreV1().Pods(metav1.NamespaceAll)
	pods, err := podResource.List(context.TODO(), metav1.ListOptions{Limit: 500})
	panicErr(err)

	fmt.Println("NAMESPACE, \t\t\t\t NAME \t\t\t STATUS")
	for _, pod := range pods.Items {
		fmt.Printf("%v \t\t\t %v \t\t\t %+v\n", pod.Namespace, pod.Name, pod.Status.Phase)
	}
}
