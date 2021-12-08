/*
Copyright 2021 Anjie.

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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ModelBoxSpec defines the desired state of ModelBox
type ModelBoxSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Name is an example field of ModelBox. Edit modelbox_types.go to remove/update
	Name           string                      `json:"name,omitempty"`           // 服务名称
	Image          string                      `json:"image,omitempty"`          // 镜像
	Replicas       *int32                      `json:"replicas,omitempty"`       // 副本数
	ModelFileURL   string                      `json:"modelFileURL,omitempty"`   // 模型文件
	ServiceType    corev1.ServiceType          `json:"serviceType,omitempty"`    // 服务类型
	Ports          []corev1.ServicePort        `json:"ports"`                    // 服务端口
	Resources      corev1.ResourceRequirements `json:"resources,omitempty"`      // 资源配额
	ResourceType   string                      `json:"resourceType,omitempty"`   // 资源规格
	Envs           []corev1.EnvVar             `json:"envs,omitempty"`           // 环境变量
	RollingUpdate  string                      `json:"rollingUpdate"`            // 配置滚动更新百分比
	ReadinessProbe *corev1.Probe               `json:"readinessProbe,omitempty"` // 就绪探针
	LivenessProbe  *corev1.Probe               `json:"livenessProbe,omitempty"`  // 存活探针
}

// ModelBoxStatus defines the observed state of ModelBox
// 描述app的状态信息
type ModelBoxStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	appsv1.DeploymentStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ModelBox is the Schema for the modelboxes API
type ModelBox struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ModelBoxSpec   `json:"spec,omitempty"`
	Status ModelBoxStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ModelBoxList contains a list of ModelBox
type ModelBoxList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ModelBox `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ModelBox{}, &ModelBoxList{})
}
