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
	Image string `json:"image"`
	// service占用的宿主机端口，外部请求通过此端口访问pod的服务
	Port *int32 `json:"port"`

	// 单个pod的QPS上限
	SinglePodQPS *int32 `json:"singlePodQPS"`
	// 当前整个业务的总QPS
	TotalQPS *int32 `json:"totalQPS"`
}

// ExportPortStatus defines the observed state of ExportPort
type ExportPortStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	RealQPS *int32 `json:"realQPS"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ExportPort is the Schema for the exportports API
type ExportPort struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExportPortSpec   `json:"spec,omitempty"`
	Status ExportPortStatus `json:"status,omitempty"`
}

func (in *ExportPort) String() string {
	var realQPS string

	if nil == in.Status.RealQPS {
		realQPS = "nil"
	} else {
		realQPS = strconv.Itoa(int(*(in.Status.RealQPS)))
	}

	return fmt.Sprintf("Image [%s], Port [%d], SinglePodQPS [%d], TotalQPS [%d], RealQPS [%s]",
		in.Spec.Image,
		*(in.Spec.Port),
		*(in.Spec.SinglePodQPS),
		*(in.Spec.TotalQPS),
		realQPS)
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
