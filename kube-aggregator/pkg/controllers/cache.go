package controllers

import (
	"fmt"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

// WaitForCacheSync is wrapper around cache.WaitForCacheSync that generates log messages indicating that controller
// identified by controllerName is waiting for syncs, followed by either a successful or failed sync.
func WaitForCacheSync(controllerName string, stopCh <-chan struct{}, cacheSyncs ...cache.InformerSynced) bool {
	klog.Infof("Waiting for caches to sync for %s controller", controllerName)

	if !cache.WaitForCacheSync(stopCh, cacheSyncs...) {
		utilruntime.HandleError(fmt.Errorf("unable to sync caches for %s controller", controllerName))
		return false
	}

	klog.Infof("Cache are synced for %s controller.", controllerName)
	return true
}
