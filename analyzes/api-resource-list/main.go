package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	apiResourceList := []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{
					Name:       "pods",
					Namespaced: true,
					Kind:       "Pod",
					Verbs: metav1.Verbs{
						"get", "list", "delete", "deletecolletction", "create", "update", "patch", "watch",
					},
				},
				{
					Name:       "services",
					Namespaced: true,
					Kind:       "Service",
					Verbs: metav1.Verbs{
						"get", "list", "delete", "deletecolletction", "create", "update",
					},
				},
			},
		},
		{
			GroupVersion: "apps/v1",
			APIResources: []metav1.APIResource{
				{
					Name:       "deployments",
					Namespaced: true,
					Kind:       "Deployment",
					Verbs: metav1.Verbs{
						"get", "list", "delete", "deletecolletction", "create", "update",
					},
				},
			},
		},
	}
	for _, resourceList := range apiResourceList {
		println(resourceList.GroupVersion)
		for _, resource := range resourceList.APIResources {
			println(resource.String())
		}
	}
}
