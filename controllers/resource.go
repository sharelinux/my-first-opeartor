package controllers

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
					Volumes:        newVolumes(modelbox),
				},
			},
			Selector: selector,
		},
	}
}

func newVolumes(modelbox *modelv1.ModelBox) []corev1.Volume {
	var volumes []corev1.Volume
	// 模型下载存储空间
	modelFileVolume := corev1.Volume{
		Name: "model-volume",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	volumes = append(volumes, modelFileVolume)

	// localtime
	localFileVolume := corev1.Volume{
		Name: "localtime",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/etc/localtime",
			},
		},
	}
	volumes = append(volumes, localFileVolume)

	return volumes
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
			Type: modelbox.Spec.ServiceType,
			//Type: corev1.ServiceTypeNodePort,
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
		Resources: newResourceRequirements(modelbox),
		Env:       modelbox.Spec.Envs,
		Ports:     containerPorts,
		//Command: []string{"start"},
		//ReadinessProbe: newReadinessProbe(modelbox), // 注入就绪探针，检测成功就关联svc
		//LivenessProbe:  newLivenessProbe(modelbox),  // 注入存活探针，检测失败就重启或者终止该容器
		VolumeMounts: []corev1.VolumeMount{
			corev1.VolumeMount{
				Name:      "model-volume",
				MountPath: "/app/model",
			},
		},
	})

	// 添加一个通用容器
	containers = append(containers, corev1.Container{
		Name:      "db-container",
		Image:     "busybox",
		Command:   []string{"/bin/sh", "-c", "sleep 86400"},
		Resources: newResourceRequirements(modelbox),
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
		Name:  "init-container",
		Image: "busybox",
		Command: []string{"/bin/sh", "-c", "sleep 3 && touch /tmp/done"},
		//Command: []string{"/root/s3client"},
		//Resources: modelbox.Spec.Resources,
		Resources: newResourceTypeRequirements("small"),
		Env:       modelbox.Spec.Envs,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "model-volume",
				MountPath: "/app/model",
			},
		},
	})

	return containers
}

func newResourceRequirements(modelbox *modelv1.ModelBox) corev1.ResourceRequirements {
	var rr corev1.ResourceRequirements

	resourceType := modelbox.Spec.ResourceType
	switch resourceType {
	case "small":
		rr = newResourceTypeRequirements(resourceType)
	case "medium":
		rr = newResourceTypeRequirements(resourceType)
	case "large":
		rr = newResourceTypeRequirements(resourceType)
	case "custom":
		rr = modelbox.Spec.Resources
	}

	return rr
}

func newResourceTypeRequirements(resourceType string) corev1.ResourceRequirements {
	var rr corev1.ResourceRequirements
	var resourceCPU string
	var resourceMemory string

	switch resourceType {
	case "small":
		resourceCPU = "1000m"
		resourceMemory = "2Gi"
	case "medium":
		resourceCPU = "2000m"
		resourceMemory = "4Gi"
	case "large":
		resourceCPU = "4000m"
		resourceMemory = "8Gi"
	}

	rr.Limits = corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse(resourceCPU),
		corev1.ResourceMemory: resource.MustParse(resourceMemory),
	}
	rr.Requests = corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse(resourceCPU),
		corev1.ResourceMemory: resource.MustParse(resourceMemory),
	}

	return rr
}

func newReadinessProbe(modelbox *modelv1.ModelBox) *corev1.Probe {
	if modelbox.Spec.ReadinessProbe != nil {
		return modelbox.Spec.ReadinessProbe
	}
	//readinessProbe:
	//	initialDelaySeconds: 20
	//	periodSeconds: 5
	//	timeoutSeconds: 10
	//	httpGet:
	//		scheme: HTTP
	//		port: 8081
	//		path: /actuator/health
	return &corev1.Probe{
		InitialDelaySeconds: 10,
		PeriodSeconds:       5,
		TimeoutSeconds:      10,
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Scheme: corev1.URISchemeHTTP,
				Port:   intstr.FromInt(8080),
				Path:   "/healthz",
			},
		},
	}
}

func newLivenessProbe(modelbox *modelv1.ModelBox) *corev1.Probe {
	if modelbox.Spec.LivenessProbe != nil {
		return modelbox.Spec.LivenessProbe
	}

	//livenessProbe:
	//  initialDelaySeconds: 30
	//  periodSeconds: 10
	//  timeoutSeconds: 5
	//  httpGet:
	//    scheme: HTTP
	//    port: 8081
	//    path: /health

	return &corev1.Probe{
		InitialDelaySeconds: 10,
		PeriodSeconds:       5,
		TimeoutSeconds:      10,
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Scheme: corev1.URISchemeHTTP,
				Port:   intstr.FromInt(8080),
				Path:   "/healthz",
			},
		},
	}
}
