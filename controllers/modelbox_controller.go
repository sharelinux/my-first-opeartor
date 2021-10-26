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

package controllers

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/util/retry"
	"reflect"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	modelv1 "github.com/sharelinuxs/my-first-opeartor/api/v1"
)

var (
	oldSpecAnnotation = "modelbox.model.github.com/last-oldSpec"
)

// ModelBoxReconciler reconciles a ModelBox object
type ModelBoxReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=model.github.com,resources=modelboxes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=model.github.com,resources=modelboxes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=model.github.com,resources=modelboxes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ModelBox object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
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

// SetupWithManager sets up the controller with the Manager.
func (r *ModelBoxReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&modelv1.ModelBox{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
