package v1

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilsPtr "k8s.io/utils/ptr"
)

func addDefaultingFunctions(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// SetDefaults_ServiceReference sets defaults for AuditSync Webhook's ServiceReference
//
//goland:noinspection GoSnakeCaseUsage
func SetDefaults_ServiceReference(obj *ServiceReference) {
	if obj.Port == nil {
		obj.Port = utilsPtr.To[int32](443)
	}
}
