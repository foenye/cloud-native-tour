/*
Copyright 2025 Yeahfo.

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

package controller

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	applicationv1 "github.com/yeahfo/cloud-native-tour/application-operator/api/v1"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=apps.yeahfo.github.io,resources=applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.yeahfo.github.io,resources=applications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.yeahfo.github.io,resources=applications/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Application object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (reconciled *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Gets the Application
	app := &applicationv1.Application{}

	if err := reconciled.Get(ctx, req.NamespacedName, app); errors.IsNotFound(err) {
		logger.Info("The Application is not found.")
		return ctrl.Result{}, nil
	} else if err != nil {
		logger.Error(err, "Failed to get the Application.")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}
	// Creates pods
	for i := range app.Spec.Replicas {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%d", app.Name, i),
				Namespace: app.Namespace,
				Labels:    app.Labels,
			},
			Spec: app.Spec.Template.Spec,
		}

		if err := reconciled.Create(ctx, pod); err != nil {
			logger.Error(err, "Failed to create Pod")
			return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
		}
		logger.Info(fmt.Sprintf("The Pod (%s) has created.", pod.Name))
	}
	logger.Info("All pods has created.")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (reconciled *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&applicationv1.Application{}).
		Named("application").
		Complete(reconciled)
}
