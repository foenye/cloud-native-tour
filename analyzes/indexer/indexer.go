package main

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"strings"
)

func main() {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{"byUser": UsersIndexFn})
	pod1 := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:        "one",
		Annotations: map[string]string{"users": "ernie,bert"},
	}}
	pod2 := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:        "two",
		Annotations: map[string]string{"users": "bert,oscar"},
	}}
	pod3 := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:        "tre",
		Annotations: map[string]string{"users": "ernie,elmo"},
	}}

	_ = indexer.Add(pod1)
	_ = indexer.Add(pod2)
	_ = indexer.Add(pod3)

	erniePods, err := indexer.ByIndex("byUser", "bert")
	if err != nil {
		panic(err)
	}

	for _, erniePod := range erniePods {
		fmt.Println(erniePod.(*corev1.Pod).Name)
	}
}

func UsersIndexFn(obj interface{}) ([]string, error) {
	pod := obj.(*corev1.Pod)
	usersString := pod.Annotations["users"]
	return strings.Split(usersString, ","), nil
}
