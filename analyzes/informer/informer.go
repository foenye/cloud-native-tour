package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"time"
)

func PanicErr(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {

	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
	PanicErr(err)

	clientset, err := kubernetes.NewForConfig(config)
	PanicErr(err)

	stopCh := make(chan struct{})
	defer close(stopCh)
	sharedInformerFactory := informers.NewSharedInformerFactory(clientset, time.Minute)
	informer := sharedInformerFactory.Core().V1().Pods().Informer()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerDetailedFuncs{
		AddFunc: func(obj interface{}, isInInitialList bool) {
			metaObj := obj.(metav1.Object)
			log.Printf("New Pod Added to Store: %s", metaObj.GetName())
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldMetaObj := oldObj.(metav1.Object)
			newMetaObj := newObj.(metav1.Object)
			log.Printf("%s Pod Updated to %s", oldMetaObj.GetName(), newMetaObj.GetName())
		},
		DeleteFunc: func(obj interface{}) {
			metaObj := obj.(metav1.Object)
			log.Printf("Pod Deleted form Store: %s", metaObj.GetName())
		},
	})
	informer.Run(stopCh)
}
