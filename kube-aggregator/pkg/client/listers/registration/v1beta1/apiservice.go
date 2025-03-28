/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by lister-gen. DO NOT EDIT.

package v1beta1

import (
	registrationv1beta1 "github.com/foenye/cloud-native-tour/kube-aggregator/pkg/apis/registration/v1beta1"
	labels "k8s.io/apimachinery/pkg/labels"
	listers "k8s.io/client-go/listers"
	cache "k8s.io/client-go/tools/cache"
)

// APIServiceLister helps list APIServices.
// All objects returned here must be treated as read-only.
type APIServiceLister interface {
	// List lists all APIServices in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*registrationv1beta1.APIService, err error)
	// Get retrieves the APIService from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*registrationv1beta1.APIService, error)
	APIServiceListerExpansion
}

// aPIServiceLister implements the APIServiceLister interface.
type aPIServiceLister struct {
	listers.ResourceIndexer[*registrationv1beta1.APIService]
}

// NewAPIServiceLister returns a new APIServiceLister.
func NewAPIServiceLister(indexer cache.Indexer) APIServiceLister {
	return &aPIServiceLister{listers.New[*registrationv1beta1.APIService](indexer, registrationv1beta1.Resource("apiservice"))}
}
