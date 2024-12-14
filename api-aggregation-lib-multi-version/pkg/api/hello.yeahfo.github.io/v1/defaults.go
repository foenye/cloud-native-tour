package v1

import hellov1 "github.com/yeahfo/cloud-native-tour/api/hello.yeahfo.github.io/v1"

const AnnotationImage = "spec.image"

func init() {
	localSchemeBuilder.Register(RegisterDefaults)
}

// SetDefaultsFoo sets defaults for Foo
func SetDefaultsFoo(foo *hellov1.Foo) {
	if foo.Labels == nil {
		foo.Labels = map[string]string{}
	}
	if foo.Annotations == nil {
		foo.Annotations = map[string]string{}
	}
	foo.Labels[hellov1.Group+"/metadata.name"] = foo.Name
	foo.Annotations[AnnotationImage] = "busybox:1.36"
}
