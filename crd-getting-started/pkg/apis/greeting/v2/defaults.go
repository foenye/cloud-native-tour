package v2

import "k8s.io/apimachinery/pkg/runtime"

func AddDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// SetDefaults_Foo sets defaults for Foo
//
//goland:noinspection GoUnusedExportedFunction,GoSnakeCaseUsage
func SetDefaults_Foo(obj *Foo) {
	if obj.Labels == nil {
		obj.Labels = map[string]string{}
	}
	obj.Labels["greeting.foen.ye/metadata.name"] = obj.Name
}
