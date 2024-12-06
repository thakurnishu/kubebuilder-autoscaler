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
	"time"

	prometheusapi "github.com/prometheus/client_golang/api"
	prometheusapiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prometheusmodel "github.com/prometheus/common/model"

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
// +kubebuilder:rbac:groups="",resources=pods;services;endpoints;secrets;external,verbs=get;list;watch
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

	reqLogger.Info("Reconcilling ScaledObject Starts")

	for _, trigger := range scaledObject.Spec.Triggers {
		if trigger.Type == "prometheus" {

			serverAddr := trigger.Metadata.ServerAddress
			query := trigger.Metadata.Query
			err := runPrometheusQuery(ctx, serverAddr, query)
			if err != nil {
				return ctrl.Result{}, err
			}

		} else {
			reqLogger.Error(fmt.Errorf(""), "currently only support prometheus as trigger")
			return ctrl.Result{}, nil
		}
	}

	err = r.updateDeploymentReplica(ctx, scaledObject)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Error(err, fmt.Sprintf("unable to find deployment name %s in namespace %s", scaledObject.Spec.ScaleTargetRef.Name, scaledObject.Spec.ScaleTargetRef.Namespace))
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}
	reqLogger.Info("Reconcilling ScaledObject Complete", "deploymentName", scaledObject.Spec.ScaleTargetRef.Name)

	return ctrl.Result{}, nil
}

func runPrometheusQuery(ctx context.Context, serverAddr, query string) error {
	reqLogger := log.FromContext(ctx)
	// Create a Prometheus client
	client, err := prometheusapi.NewClient(prometheusapi.Config{Address: serverAddr})
	if err != nil {
		reqLogger.Error(err, "Prometheus: creating client:")
		return err
	}

	// Create the API client
	apiClient := prometheusapiv1.NewAPI(client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, warnings, err := apiClient.Query(ctx, query, time.Now())
	if err != nil {
		reqLogger.Error(err, "Prometheus: executing query:")
		return err
	}

	// Print warnings, if any
	if len(warnings) > 0 {
		reqLogger.Info("WARN: Prometheus warnings", "warnings", warnings)
	}

	vector, ok := result.(prometheusmodel.Vector)
	if !ok {
		reqLogger.Error(fmt.Errorf(""), "Prometheus Query result is not a vector.")
		return fmt.Errorf("Query result is not a vector.")
	}

	// Extract the value
	if len(vector) > 0 {
		value := vector[0].Value
		reqLogger.Info("Prometheus: ", "Result: ", value)
		fmt.Println("Result:", value)
	} else {
		reqLogger.Error(fmt.Errorf(""), "Prometheus No Data Found.")
	}
	return nil
}

func (r *ScaledObjectReconciler) updateDeploymentReplica(ctx context.Context, scaledObject *scalerv1alpha1.ScaledObject) error {
	reqLogger := log.FromContext(ctx)
	deployment := &appsv1.Deployment{}

	err := r.Get(ctx, client.ObjectKey{
		Namespace: scaledObject.Spec.ScaleTargetRef.Namespace,
		Name:      scaledObject.Spec.ScaleTargetRef.Name,
	}, deployment)
	if err != nil {
		return err
	}

	reqLogger.Info("Updating replicas to 0 for deployment", "deployment", scaledObject.Spec.ScaleTargetRef.Name)

	// Set replica to 0
	var zero int32 = 0
	deployment.Spec.Replicas = ptr.To(zero)

	// update deployment
	err = r.Update(ctx, deployment)
	if err != nil {
		reqLogger.Error(err, fmt.Sprintf("failed to update deployment replicas to 0 for deployment %s", scaledObject.Spec.ScaleTargetRef.Name))
		return err
	}
	reqLogger.Info("Successfully updated deployment replicas to 0", "deployment", scaledObject.Spec.ScaleTargetRef.Name)
	return nil
}
