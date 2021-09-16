/*
Copyright 2021.

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

package v1

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ExportPortSpec defines the desired state of ExportPort
type ExportPortSpec struct {
	// 业务服务对应的镜像，包括名称:tag
	//+kubebuilder:validation:Required
	Label string `json:"label"`

	//+kubebuilder:validation:Required
	Port *int32 `json:"port"`

	Ver string `json:"ver"`
}

// ExportPortStatus defines the observed state of ExportPort
type ExportPortStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	CurrNum *int32 `json:"currNum"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:JSONPath=".status.currNum",name=currNum,type=string

// ExportPort is the Schema for the exportports API
type ExportPort struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExportPortSpec   `json:"spec,omitempty"`
	Status ExportPortStatus `json:"status,omitempty"`
}

func (in *ExportPort) String() string {
	var currNum string

	if nil == in.Status.CurrNum {
		currNum = "0"
	} else {
		currNum = strconv.Itoa(int(*(in.Status.CurrNum)))
	}

	return fmt.Sprintf("Lable:[%s], CurrNum [%s]",
		in.Spec.Label,
		currNum)
}

//+kubebuilder:object:root=true

// ExportPortList contains a list of ExportPort
type ExportPortList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ExportPort `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ExportPort{}, &ExportPortList{})
}
