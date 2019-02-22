/*
Copyright 2019 Weiwei Zhang.

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

// LoadBalancerControllerSpec defines the desired state of LoadBalancerController
type LoadBalancerControllerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Services []LoadBalancerService `json:"services,omitempty"`
}

// LoadBalancerService defines each service has been created as Type: LoadBalancer
type LoadBalancerService struct {
	// INSERT ADDITIONAL SPEC FIELDS - load balancer type service name
	// Important: Run "make" to regenerate code after modifying this file
	Name       string      `json:"name,omitempty"`
	SyncPeriod metav1.Time `json:"syncPeriod,omitempty`
}

// LoadBalancerControllerStatus defines the observed state of LoadBalancerController
type LoadBalancerControllerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	LoadBalancerServiceStatus []ServiceStatus `json:"serviceStatus,omitempty"`
}

// ServiceStatus defines the status of each load balancer service syncing with route53 record
type ServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of service status
	// Important: Run "make" to regenerate code after modifying this file
	LastUpdateTime metav1.Time `json:"lastUpdate,omitempty"`
	Name           string      `json:"name,omitempty"`
	UpdateCount    int32       `json:"updateCount,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LoadBalancerController is the Schema for the loadbalancercontrollers API
// +k8s:openapi-gen=true
type LoadBalancerController struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LoadBalancerControllerSpec   `json:"spec,omitempty"`
	Status LoadBalancerControllerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LoadBalancerControllerList contains a list of LoadBalancerController
type LoadBalancerControllerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LoadBalancerController `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LoadBalancerController{}, &LoadBalancerControllerList{})
}
