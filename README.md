## 我的第一个k8s Operator
> 该项目是学习一下k8s的CRD + Controller开发业务控制器的简单逻辑实现。

**支持功能**:
1. 支持自定义资关联创建Deployment、Service。
2. 支持非自助删除Deployment、Service、Pod资源的自动创建和监视。
3. 支持多Container和InitContainer的注入。
4. 支持设置滚动更新的比例。
5. 支持基于语义的资源规则限制，small、medium、large、custom等
6. 支持Service自定义映射, ClusterIP、NodePort等。
7. 支持注入默认的服务存活探针和就绪探针的检测功能。

### 基于kubebuilder脚手架创建自己的Operator代码框架

```shell
# 准备相关目录
$ mkdir github.com/sharelinuxs/my-first-opeartor
$ cd github.com/sharelinuxs/my-first-opeartor

# 开启 go modules
$ export GO111MODULE=on
$ export GOPROXY=https://goproxy.cn

# 初始化项目
$ kubebuilder init --domain github.com --owner Anjie --repo github.com/sharelinuxs/my-first-opeartor

# 创建API
# Api --> CRD 映射关系
# API --> CRD model/v1 ModelBox
$ kubebuilder create api --group model --version v1 --kind ModelBox
Create Resource [y/n]
y
Create Controller [y/n]
y
Writing scaffold for you to edit...
api/v1/modelbox_types.go
controllers/modelbox_controller.go
```

### 设计实现自定义资源和控制器

#### 抽象自己的资源
```yaml
apiVersion: model.github.com/v1
kind: ModelBox
metadata:
  name: modelbox-sample
spec:
  # Add fields here
  name: "nginx"         # 名称
  image: "nginx:1.7.9"  # 镜像
  replicas: 4           # 副本数
  rollingUpdate: 30%    # 滚动升级百分比
  ports:                # 端口映射
  - port: 80
    targetPort: 80
    nodePort: 30010
```

#### CRD资源映射为API类型
```go
// api/v1/modelbox_types.go
// ModelBox is the Schema for the modelboxes API
type ModelBox struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ModelBoxSpec   `json:"spec,omitempty"`
	Status ModelBoxStatus `json:"status,omitempty"`
}

// ModelBoxSpec defines the desired state of ModelBox
type ModelBoxSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Name is an example field of ModelBox. Edit modelbox_types.go to remove/update
	Name          string                      `json:"name,omitempty"`          // 服务名称
	Image         string                      `json:"image,omitempty"`         // 镜像
	Replicas      *int32                      `json:"replicas,omitempty"`      // 副本数
	Ports         []corev1.ServicePort        `json:"ports"`                   // 服务端口
	Resources     corev1.ResourceRequirements `json:"resources,omitempty"`     // 资源配额
	Envs          []corev1.EnvVar             `json:"envs,omitempty"`          // 环境变量
	RollingUpdate string                      `json:"rollingUpdate,omitempty"` // 配置滚动更新百分比
}

```

#### 编写ModelBox 控制器具体逻辑
```go
// controllers/modelbox_controller.go
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *ModelBoxReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("modelbox", req.NamespacedName)

	// TODO: your logic here
	log.Info("modelbox reconciling")

	// 1. 首先获取 modelbox 实例
	var modelBoxInstance modelv1.ModelBox
	err := r.Get(ctx, req.NamespacedName, &modelBoxInstance)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "unable to get modelbox instance")
			// 获取失败重新入队, 并且返回错误
			return ctrl.Result{}, err
		}
		// 在删除一个不存在的对象的时候，可能会报not-found的错误。
		// 这种情况不需要重新入队列修复，直接忽略处理即可。
		return ctrl.Result{}, nil
	}

	// 当前对象标记为了删除, 无需处理
	if modelBoxInstance.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	// 2、如果不存在关联的资源，是不是应该去创建
	// 如果存在关联的资源，是不是要判断是否需要更新
	deploy := &appsv1.Deployment{}
	if err := r.Get(ctx, req.NamespacedName, deploy); err != nil && errors.IsNotFound(err) {
		// 关联Annotations
		data, err := json.Marshal(modelBoxInstance.Spec)
		if err != nil {
			return ctrl.Result{}, err
		}
		if modelBoxInstance.Annotations != nil {
			modelBoxInstance.Annotations[oldSpecAnnotation] = string(data)
		} else {
			modelBoxInstance.Annotations = map[string]string{
				oldSpecAnnotation: string(data),
			}
		}
		// 重新更新modelBoxInstance
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return r.Update(ctx, &modelBoxInstance)
		}); err != nil {
			return ctrl.Result{}, err
		}

		// Deployment 不存在，创建关联的资源
		newDeploy := NewDeploy(&modelBoxInstance)
		if err := r.Create(ctx, newDeploy); err != nil {
			r.Log.Error(err, "create deployment error")
			// 重新入队列，重试一次。
			return ctrl.Result{}, err
		}

		// 判断Service是否存在，不存在直接创建 Service
		newService := NewService(&modelBoxInstance)
		if err := r.Create(ctx, newService); err != nil {
			r.Log.Error(err, "create service error")
			// 重新入队列，重试一次。
			return ctrl.Result{}, err
		}

		// 创建成功，直接返回
		return ctrl.Result{}, nil
	}

	log.Info("modelbox instance ", "image:", modelBoxInstance.Spec.Image, "name:", modelBoxInstance.Name)

	// Todo: 更新逻辑, 是不是应该需要判断是否需要更新 (yaml文件是否发生了变化)
	// 旧的配置文件可以从annotations中获取，需要在创建资源清单的时候，就把当前配置写入注解中。
	// old Yaml --> Yaml Diff
	oldSpec := modelv1.ModelBoxSpec{}
	if err := json.Unmarshal([]byte(modelBoxInstance.Annotations[oldSpecAnnotation]), &oldSpec); err != nil {
		// 获取上一个版本配置失败，重新入队列，重试一次
		return ctrl.Result{}, err
	}

	// 是不是就可以来和新旧的对象进行比较，如果不一致是不是就应该更新。
	if !reflect.DeepEqual(modelBoxInstance.Spec, oldSpec) {
		// 应该去更新关联资源
		newDeploy := NewDeploy(&modelBoxInstance)
		oldDeploy := &appsv1.Deployment{}
		if err := r.Get(ctx, req.NamespacedName, oldDeploy); err != nil {
			// 如果查询失败，再次尝试一次查询
			return ctrl.Result{}, err
		}
		// 此处并非是删除oldDeploy 而是用newDeploy.Spec替换OldDeploySpec
		// 然后直接更新oldDeploy即可。
		oldDeploy.Spec = newDeploy.Spec
		// 注意，一般情况不会直接调用Update进行更新,防止资源被其他控制器锁定更新操作，使用重试机制进行更新。
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return r.Update(ctx, oldDeploy)
		}); err != nil {
			return ctrl.Result{}, err
		}

		// 更新: Service,
		newService := NewService(&modelBoxInstance)
		oldService := &corev1.Service{}
		if err := r.Get(ctx, req.NamespacedName, oldService); err != nil {
			// 如果查询失败，再次尝试一次查询
			return ctrl.Result{}, err
		}
		// Todo: 判断Ports是否有变化, 目前暴力实现整体覆盖更新
		newService.Spec.ClusterIP = oldService.Spec.ClusterIP
		oldService.Spec = newService.Spec
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return r.Update(ctx, oldService)
		}); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}


// controllers/resource.go
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
```

### 常用命令

#### 安装CRD
```make install```

#### 本地编译运行控制器
```shell
make run ENABLE_WEBHOOKS=false
2021-10-26T14:49:54.717+0800    INFO    controller-runtime.metrics      metrics server is starting to listen    {"addr": ":8080"}
2021-10-26T14:49:54.717+0800    INFO    setup   starting manager
2021-10-26T14:49:54.718+0800    INFO    controller-runtime.manager      starting metrics server {"path": "/metrics"}
2021-10-26T14:49:54.718+0800    INFO    controller-runtime.manager.controller.modelbox  Starting EventSource    {"reconciler group": "model.github.com", "reconciler kind": "ModelBox", "source": "kind source: /, Kind="}
2021-10-26T14:49:54.818+0800    INFO    controller-runtime.manager.controller.modelbox  Starting EventSource    {"reconciler group": "model.github.com", "reconciler kind": "ModelBox", "source": "kind source: /, Kind="}
2021-10-26T14:49:54.919+0800    INFO    controller-runtime.manager.controller.modelbox  Starting EventSource    {"reconciler group": "model.github.com", "reconciler kind": "ModelBox", "source": "kind source: /, Kind="}
2021-10-26T14:49:55.020+0800    INFO    controller-runtime.manager.controller.modelbox  Starting Controller     {"reconciler group": "model.github.com", "reconciler kind": "ModelBox"}
2021-10-26T14:49:55.020+0800    INFO    controller-runtime.manager.controller.modelbox  Starting workers        {"reconciler group": "model.github.com", "reconciler kind": "ModelBox", "worker count": 1}
```

#### 创建资源
```kubectl apply -f config/samples/model_v1_modelbox.yaml```

#### 删除CRD
```make uninstall```
