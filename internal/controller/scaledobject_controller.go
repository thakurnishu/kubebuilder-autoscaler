/*
Copyright 2024 Nishant Singh.

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

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	scalerv1alpha1 "github.com/thakurnishu/kubebuilder-autoscaler/api/v1alpha1"
)

// ScaledObjectReconciler reconciles a ScaledObject object
type ScaledObjectReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScaledObjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&scalerv1alpha1.ScaledObject{}).
		Named("scaledobject").
		Complete(r)
}

// +kubebuilder:rbac:groups=scaler.nishantsingh.sh,resources=scaledobjects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=scaler.nishantsingh.sh,resources=scaledobjects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=scaler.nishantsingh.sh,resources=scaledobjects/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;update;patch;create;delete
// +kubebuilder:rbac:groups="",resources=pods;services;services;secrets;external,verbs=get;list;watch
// +kubebuilder:rbac:groups="*",resources="*",verbs=get
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ScaledObject object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *ScaledObjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)

	// TODO(user): your logic here
	scaledObject := &scalerv1alpha1.ScaledObject{}
	err := r.Get(ctx, req.NamespacedName, scaledObject)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		// Error Reading the object - requeue the request
		reqLogger.Error(err, "failed to get ScaledObject")
		return ctrl.Result{}, err
	}

	reqLogger.Info("Reconcilling ScaledObject Starts", "deploymentName", scaledObject.Spec.DeploymentName, "namespace", scaledObject.Namespace)
	err = r.updateDeploymentReplica(ctx, scaledObject)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Error(err, fmt.Sprintf("unable to find deployment name %s in namespace %s", scaledObject.Spec.DeploymentName, scaledObject.Spec.Namespace))
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}
	reqLogger.Info("Reconcilling ScaledObject Complete", "deploymentName", scaledObject.Spec.DeploymentName)

	// Check if the ScaledObject instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	//if scaledObject.GetDeletionTimestamp() != nil {
	//    return ctrl.Result{}, err // TODO
	//}
	return ctrl.Result{}, nil
}

func (r *ScaledObjectReconciler) updateDeploymentReplica(ctx context.Context, scaledObject *scalerv1alpha1.ScaledObject) error {
	reqLogger := log.FromContext(ctx)
	deployment := &appsv1.Deployment{}

	err := r.Get(ctx, client.ObjectKey{
		Namespace: scaledObject.Spec.Namespace,
		Name:      scaledObject.Spec.DeploymentName,
	}, deployment)
	if err != nil {
		return err
	}

	reqLogger.Info("Updating replicas to 0 for deployment", "deployment", scaledObject.Spec.DeploymentName)

	// Set replica to 0
	var zero int32 = 0
	deployment.Spec.Replicas = ptr.To(zero)

	// update deployment
	err = r.Update(ctx, deployment)
	if err != nil {
		reqLogger.Error(err, fmt.Sprintf("failed to update deployment replicas to 0 for deployment %s", scaledObject.Spec.DeploymentName))
		return err
	}
	reqLogger.Info("Successfully updated deployment replicas to 0", "deployment", scaledObject.Spec.DeploymentName)
	return nil
}
