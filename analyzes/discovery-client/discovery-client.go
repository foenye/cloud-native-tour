package main

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
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

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	panicErr(err)

	_, apiResourceList, err := discoveryClient.ServerGroupsAndResources()
	panicErr(err)

	for _, list := range apiResourceList {
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		panicErr(err)
		for _, resource := range list.APIResources {
			fmt.Printf("name: %v, group: %v, version: %v \n", resource.Name, gv.Group, gv.Version)
		}

	}
}
