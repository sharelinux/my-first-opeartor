package controllers

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"

	modelv1 "github.com/sharelinuxs/my-first-opeartor/api/v1"
)

// NewDeploy 创建 modelbox的 kubernetes Deployment
func NewDeploy(modelbox *modelv1.ModelBox) *appsv1.Deployment {
	labels := map[string]string{"modelbox": modelbox.Name}
	selector := &metav1.LabelSelector{
		MatchLabels: labels,
	}
	maxUnavailable := intstr.FromString(modelbox.Spec.RollingUpdate)
	maxSurge := intstr.FromString(modelbox.Spec.RollingUpdate)
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      modelbox.Name,
			Namespace: modelbox.Namespace,
			// 主要的属性 OwnerReferences 如果删除Modelbox，就需要自动关联删除Deployment、Service资源
			OwnerReferences: makeOwnerReferences(modelbox),
		},
		Spec: appsv1.DeploymentSpec{
			MinReadySeconds: int32(3), // 最小等待可用秒数
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &maxUnavailable,
					MaxSurge:       &maxSurge,
				},
			},
			Replicas: modelbox.Spec.Replicas,
			Template: corev1.PodTemplateSpec{ // Pod Template
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					InitContainers: newInitContainers(modelbox),
					Containers:     newContainers(modelbox),
				},
			},
			Selector: selector,
		},
	}
}

// NewService 创建 modelbox 的 kubernetes Service
func NewService(modelbox *modelv1.ModelBox) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      modelbox.Name,
			Namespace: modelbox.Namespace,
			// 主要的属性 OwnerReferences 如果删除Modelbox，就需要自动关联删除Deployment、Service资源
			OwnerReferences: makeOwnerReferences(modelbox),
		},
		Spec: corev1.ServiceSpec{
			Ports: modelbox.Spec.Ports,
			// 此处可以定义为 Ingress
			Type: corev1.ServiceTypeNodePort,
			Selector: map[string]string{
				"modelbox": modelbox.Name,
			},
		},
	}
}

// newContainers 需要创建的容器组
func newContainers(modelbox *modelv1.ModelBox) []corev1.Container {
	var containers []corev1.Container
	var containerPorts []corev1.ContainerPort

	for _, svcPort := range modelbox.Spec.Ports {
		containerPorts = append(containerPorts, corev1.ContainerPort{
			ContainerPort: svcPort.TargetPort.IntVal,
		})
	}

	// 场景1: 单业务容器直接构造返回
	//return []corev1.Container{
	//	{
	//		Name: modelbox.Name,
	//		Image: modelbox.Spec.Image,
	//		Resources: modelbox.Spec.Resources,
	//		Env: modelbox.Spec.Envs,
	//		Ports: containerPorts,
	//	},
	//}

	// 场景2: 业务容器需要InitContainer 或需要注入容器
	// 添加业务 Container
	containers = append(containers, corev1.Container{
		Name:      modelbox.Name,
		Image:     modelbox.Spec.Image,
		Resources: modelbox.Spec.Resources,
		Env:       modelbox.Spec.Envs,
		Ports:     containerPorts,
	})

	// 添加一个通用容器
	containers = append(containers, corev1.Container{
		Name:      "db-container",
		Image:     "busybox",
		Command:   []string{"/bin/sh", "-c", "sleep 86400"},
		Resources: modelbox.Spec.Resources,
		Env:       modelbox.Spec.Envs,
	})

	return containers
}

// makeOwnerReferences 如果删除Modelbox，就需要自动关联删除Deployment、Service资源
func makeOwnerReferences(modelbox *modelv1.ModelBox) []metav1.OwnerReference {
	return []metav1.OwnerReference{
		// 生成一个References
		*metav1.NewControllerRef(modelbox, schema.GroupVersionKind{
			Group:   modelv1.GroupVersion.Group,
			Version: modelv1.GroupVersion.Version,
			Kind:    modelv1.Kind,
		}),
	}
}

func newInitContainers(modelbox *modelv1.ModelBox) []corev1.Container {
	var containers []corev1.Container

	// 注入 InitContainer
	containers = append(containers, corev1.Container{
		Name:      "init-container",
		Image:     "busybox",
		Command:   []string{"/bin/sh", "-c", "sleep 60 && touch /tmp/done"},
		Resources: modelbox.Spec.Resources,
		Env:       modelbox.Spec.Envs,
	})

	return containers
}
