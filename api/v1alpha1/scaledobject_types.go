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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type Trigger struct {
	Type     string          `json:"type"`
	Metadata TriggerMetadata `json:"metadata"`
}

type TriggerMetadata struct {
	// Address of Prometheus server. If using VictoriaMetrics cluster version, set full URL to Prometheus querying API, e.g. http://<vmselect>:8481/select/0/prometheus
	ServerAddress string `json:"serverAddress"`
	// Query to run. Note: query must return a vector single element response
	Query string `json:"query"`
	// Value to start scaling for (This value can be a float)
	Threshold string `json:"threshold"`
}

// ScaledObjectSpec defines the desired state of ScaledObject.
type ScaledObjectSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	ScaleTargetRef ScaleTargetRef `json:"scaleTargetRef"`
	Triggers       []Trigger      `json:"triggers"`
}

type ScaleTargetRef struct {
	// type of target [only support deployment for now]
	Type string `json:"type"`
	// name of target
	Name string `json:"name"`
	// namespace where target is
	Namespace string `json:"namespace"`
}

// ScaledObjectStatus defines the observed state of ScaledObject.
type ScaledObjectStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ScaledObject is the Schema for the scaledobjects API.
type ScaledObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScaledObjectSpec   `json:"spec,omitempty"`
	Status ScaledObjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ScaledObjectList contains a list of ScaledObject.
type ScaledObjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ScaledObject `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ScaledObject{}, &ScaledObjectList{})
}
