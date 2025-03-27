package v1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// SetDefaults_ServiceReference sets defaults for AuditSync Webhook's ServiceReference
//
//goland:noinspection GoUnusedExportedFunction,GoSnakeCaseUsage
func SetDefaults_ServiceReference(obj *ServiceReference) {
	if obj.Port == nil {
		obj.Port = ptr.To[int32](443)
	}
}
