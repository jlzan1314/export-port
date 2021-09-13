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

package controllers

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	exportportv1 "export-port/api/v1"
)

const (
	// deployment中的APP标签名
	APP_NAME = "elastic-app"
	// tomcat容器的端口号
	CONTAINER_PORT = 8080
	// 单个POD的CPU资源申请
	CPU_REQUEST = "100m"
	// 单个POD的CPU资源上限
	CPU_LIMIT = "100m"
	// 单个POD的内存资源申请
	MEM_REQUEST = "512Mi"
	// 单个POD的内存资源上限
	MEM_LIMIT = "512Mi"
)

// ExportPortReconciler reconciles a ExportPort object
type ExportPortReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=export-port.com.test,resources=exportports,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=export-port.com.test,resources=exportports/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=export-port.com.test,resources=exportports/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ExportPort object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *ExportPortReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// your logic here
	// your logic here

	log.Info("1. start reconcile logic")

	// 实例化数据结构
	instance := &exportportv1.ExportPort{}

	// 通过客户端工具查询，查询条件是
	err := r.Get(ctx, req.NamespacedName, instance)

	if err != nil {

		// 如果没有实例，就返回空结果，这样外部就不再立即调用Reconcile方法了
		if errors.IsNotFound(err) {
			log.Info("2.1. instance not found, maybe removed")
			return reconcile.Result{}, nil
		}

		log.Error(err, "2.2 error")
		// 返回错误信息给外部
		return ctrl.Result{}, err
	}

	log.Info("3. instance : " + instance.String())

	// 查找deployment
	deployment := &appsv1.Deployment{}

	// 用客户端工具查询
	err = r.Get(ctx, req.NamespacedName, deployment)

	// 查找时发生异常，以及查出来没有结果的处理逻辑
	if err != nil {
		// 如果没有实例就要创建了
		if errors.IsNotFound(err) {
			log.Info("4. deployment not exists")

			// 如果对QPS没有需求，此时又没有deployment，就啥事都不做了
			if *(instance.Spec.TotalQPS) < 1 {
				log.Info("5.1 not need deployment")
				// 返回
				return ctrl.Result{}, nil
			}

			// 先要创建service
			if err = createServiceIfNotExists(ctx, r, instance, req); err != nil {
				log.Error(err, "5.2 error")
				// 返回错误信息给外部
				return ctrl.Result{}, err
			}

			// 立即创建deployment
			if err = createDeployment(ctx, r, instance); err != nil {
				log.Error(err, "5.3 error")
				// 返回错误信息给外部
				return ctrl.Result{}, err
			}

			// 如果创建成功就更新状态
			if err = updateStatus(ctx, r, instance); err != nil {
				log.Error(err, "5.4. error")
				// 返回错误信息给外部
				return ctrl.Result{}, err
			}

			// 创建成功就可以返回了
			return ctrl.Result{}, nil
		} else {
			log.Error(err, "7. error")
			// 返回错误信息给外部
			return ctrl.Result{}, err
		}
	}

	// 如果查到了deployment，并且没有返回错误，就走下面的逻辑

	// 根据单QPS和总QPS计算期望的副本数
	expectReplicas := getExpectReplicas(instance)

	// 当前deployment的期望副本数
	realReplicas := *deployment.Spec.Replicas

	log.Info(fmt.Sprintf("9. expectReplicas [%d], realReplicas [%d]", expectReplicas, realReplicas))

	// 如果相等，就直接返回了
	if expectReplicas == realReplicas {
		log.Info("10. return now")
		return ctrl.Result{}, nil
	}

	// 如果不等，就要调整
	*(deployment.Spec.Replicas) = expectReplicas

	log.Info("11. update deployment's Replicas")
	// 通过客户端更新deployment
	if err = r.Update(ctx, deployment); err != nil {
		log.Error(err, "12. update deployment replicas error")
		// 返回错误信息给外部
		return ctrl.Result{}, err
	}

	log.Info("13. update status")

	// 如果更新deployment的Replicas成功，就更新状态
	if err = updateStatus(ctx, r, instance); err != nil {
		log.Error(err, "14. update status error")
		// 返回错误信息给外部
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExportPortReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&exportportv1.ExportPort{}).
		Complete(r)
}

// 根据单个QPS和总QPS计算pod数量
func getExpectReplicas(elasticWeb *exportportv1.ExportPort) int32 {
	// 单个pod的QPS
	singlePodQPS := *(elasticWeb.Spec.SinglePodQPS)

	// 期望的总QPS
	totalQPS := *(elasticWeb.Spec.TotalQPS)

	// Replicas就是要创建的副本数
	replicas := totalQPS / singlePodQPS

	if totalQPS%singlePodQPS > 0 {
		replicas++
	}

	return replicas
}

func createServiceIfNotExists(ctx context.Context, r *ExportPortReconciler, exportPort *exportportv1.ExportPort, req ctrl.Request) error {
	log := log.FromContext(ctx)

	service := &corev1.Service{}

	err := r.Get(ctx, req.NamespacedName, service)

	// 如果查询结果没有错误，证明service正常，就不做任何操作
	if err == nil {
		log.Info("service exists")
		return nil
	}

	// 如果错误不是NotFound，就返回错误
	if !errors.IsNotFound(err) {
		log.Error(err, "query service error")
		return err
	}

	// 实例化一个数据结构
	service = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: exportPort.Namespace,
			Name:      exportPort.Name,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:     "http",
				Port:     8080,
				NodePort: *exportPort.Spec.Port,
			},
			},
			Selector: map[string]string{
				"app": APP_NAME,
			},
			Type: corev1.ServiceTypeNodePort,
		},
	}

	// 这一步非常关键！
	// 建立关联后，删除elasticweb资源时就会将deployment也删除掉
	log.Info("set reference")
	if err := controllerutil.SetControllerReference(exportPort, service, r.Scheme); err != nil {
		log.Error(err, "SetControllerReference error")
		return err
	}

	// 创建service
	log.Info("start create service")
	if err := r.Create(ctx, service); err != nil {
		log.Error(err, "create service error")
		return err
	}

	log.Info("create service success")

	return nil
}

// 新建deployment
func createDeployment(ctx context.Context, r *ExportPortReconciler, exportPort *exportportv1.ExportPort) error {
	log := log.FromContext(ctx)

	// 计算期望的pod数量
	expectReplicas := getExpectReplicas(exportPort)

	log.Info(fmt.Sprintf("expectReplicas [%d]", expectReplicas))

	// 实例化一个数据结构
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: exportPort.Namespace,
			Name:      exportPort.Name,
		},
		Spec: appsv1.DeploymentSpec{
			// 副本数是计算出来的
			Replicas: pointer.Int32Ptr(expectReplicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": APP_NAME,
				},
			},

			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": APP_NAME,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: APP_NAME,
							// 用指定的镜像
							Image:           exportPort.Spec.Image,
							ImagePullPolicy: "IfNotPresent",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolSCTP,
									ContainerPort: CONTAINER_PORT,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse(CPU_REQUEST),
									"memory": resource.MustParse(MEM_REQUEST),
								},
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse(CPU_LIMIT),
									"memory": resource.MustParse(MEM_LIMIT),
								},
							},
						},
					},
				},
			},
		},
	}

	// 这一步非常关键！
	// 建立关联后，删除elasticweb资源时就会将deployment也删除掉
	log.Info("set reference")
	if err := controllerutil.SetControllerReference(exportPort, deployment, r.Scheme); err != nil {
		log.Error(err, "SetControllerReference error")
		return err
	}

	// 创建deployment
	log.Info("start create deployment")
	if err := r.Create(ctx, deployment); err != nil {
		log.Error(err, "create deployment error")
		return err
	}

	log.Info("create deployment success")

	return nil
}

// 完成了pod的处理后，更新最新状态
func updateStatus(ctx context.Context, r *ExportPortReconciler, exportPort *exportportv1.ExportPort) error {
	log := log.FromContext(ctx)

	// 单个pod的QPS
	singlePodQPS := *(exportPort.Spec.SinglePodQPS)

	// pod总数
	replicas := getExpectReplicas(exportPort)

	// 当pod创建完毕后，当前系统实际的QPS：单个pod的QPS * pod总数
	// 如果该字段还没有初始化，就先做初始化
	if nil == exportPort.Status.RealQPS {
		exportPort.Status.RealQPS = new(int32)
	}

	*(exportPort.Status.RealQPS) = singlePodQPS * replicas

	log.Info(fmt.Sprintf("singlePodQPS [%d], replicas [%d], realQPS[%d]", singlePodQPS, replicas, *(exportPort.Status.RealQPS)))

	if err := r.Update(ctx, exportPort); err != nil {
		log.Error(err, "update instance error")
		return err
	}

	return nil
}
