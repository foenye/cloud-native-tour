package v1

const AnnotationImage = "spec.image"

func init() {
	localSchemeBuilder.Register(RegisterDefaults)
}

// SetDefaults_Foo sets defaults for Foo
//
//goland:noinspection GoUnusedExportedFunction,GoSnakeCaseUsage
func SetDefaults_Foo(obj *Foo) {
	if obj.Labels == nil {
		obj.Labels = map[string]string{}
	}
	if obj.Annotations == nil {
		obj.Annotations = map[string]string{}
	}
	obj.Labels["greeting.foen.ye/metadata.name"] = obj.Name
	obj.Annotations[AnnotationImage] = "busybox:1.36"
}
