package main

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func main() {
	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"name": "foo"},
		},
	}

	object := runtime.Object(&pod)
	println(object.GetObjectKind().GroupVersionKind().Group)
	println(object.GetObjectKind().GroupVersionKind().Version)
	println(object.GetObjectKind().GroupVersionKind().Kind)

	if copied, casted := object.DeepCopyObject().(*corev1.Pod); !casted {
		println("cast error")
	} else {
		copied.Labels["name"] = "Kubernetes"
		println(copied.Labels["name"])
	}

}
