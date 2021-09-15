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
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	exportportv1 "export-port/api/v1"
)

const (
	// deployment中的APP标签名
	APP_NAME = "export-port-app"
	// tomcat容器的端口号
	CONTAINER_PORT = 8080
	// 单个POD的CPU资源申请
	CPU_REQUEST = "100m"
	// 单个POD的CPU资源上限
	CPU_LIMIT = "100m"
	// 单个POD的内存资源申请
	MEM_REQUEST = "512Mi"
	// 单个POD的内存资源上限
	MEM_LIMIT   = "512Mi"
	SERVICE_TAG = "export-port"
)

// ExportPortReconciler reconciles a ExportPort object
type ExportPortReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	ApiReader client.Reader
}

//+kubebuilder:rbac:groups=export-port.com.test,resources=exportports,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=export-port.com.test,resources=exportports/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=export-port.com.test,resources=exportports/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=ingress,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="apps",resources=pods,verbs=get;list;watch;create;update;patch;delete

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
	log.Info("NamespacedName:" + req.NamespacedName.String())

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

	DefaultLabelMap.Store(instance.Spec.Label, instance)

	log.Info("3. instance : " + instance.String())

	// 查找deployment
	ingress := &networkingv1.Ingress{}

	// 用客户端工具查询
	err = r.ApiReader.Get(ctx, req.NamespacedName, ingress)
	// 查找时发生异常，以及查出来没有结果的处理逻辑
	pods, err2 := r.getLabelPods(ctx, instance.Spec.Label)
	if err2 != nil {
		log.Error(err2, "getLabelPods error")
		// 返回错误信息给外部
		return ctrl.Result{}, err2
	}

	if err != nil {
		// 如果没有实例就要创建了
		if errors.IsNotFound(err) {
			log.Info("4. Ingress not exists")
			if ingress, err = createIngress(ctx, r, instance, pods, req); err != nil {
				log.Error(err, "createIngress result")
				// 返回错误信息给外部
				return ctrl.Result{}, err
			} else {
				return ctrl.Result{}, nil
			}
		} else {
			log.Error(err, "7. error")
			// 返回错误信息给外部
			return ctrl.Result{}, err
		}
	}

	ingress = insertIngress(ctx, r, pods, ingress, instance, req)
	err = r.Update(ctx, ingress)
	// 如果更新deployment的Replicas成功，就更新状态
	if err = updateStatus(ctx, r, instance, int32(len(pods.Items))); err != nil {
		return ctrl.Result{}, err
	}

	if err != nil {
		log.Error(err, "insertIngress error")
		// 返回错误信息给外部
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func insertIngress(ctx context.Context, r *ExportPortReconciler, pods *corev1.PodList, ingress *networkingv1.Ingress, exportPort *exportportv1.ExportPort, req ctrl.Request) *networkingv1.Ingress {
	log := log.FromContext(ctx).WithName("insertIngress")
	paths := make([]networkingv1.HTTPIngressPath, 0)
	var num int32
	for _, pod := range pods.Items {
		labels := pod.GetLabels()
		if _, ok := labels["export-port"]; !ok {
			labels["export-port"] = pod.Name
			pod.SetLabels(labels)
			r.Update(ctx, &pod)
		}

		log.Info(fmt.Sprintf("pod:[%s] ip:%s", pod.Name, pod.Status.PodIP))
		if err := createPodServiceIfNotExists(ctx, r, exportPort, req, &pod); err != nil {
			log.Error(err, "createPodServiceIfNotExists error")
		}

		p := networkingv1.PathTypePrefix

		paths = append(paths, networkingv1.HTTPIngressPath{
			Path: fmt.Sprintf("/%s/", pod.Name),
			//Path: "/",
			PathType: &p,
			Backend: networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: pod.Name,
					Port: networkingv1.ServiceBackendPort{Number: int32(8080)},
				},
			},
		})
		num++
	}

	ingress.Spec.Rules = []networkingv1.IngressRule{
		{
			IngressRuleValue: networkingv1.IngressRuleValue{
				HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: paths,
				},
			},
		},
	}

	return ingress
}

func createIngress(ctx context.Context, r *ExportPortReconciler, exportPort *exportportv1.ExportPort, pods *corev1.PodList, req ctrl.Request) (*networkingv1.Ingress, error) {
	log := log.FromContext(ctx)
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: exportPort.Namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "/",
			},
		},
		Spec: networkingv1.IngressSpec{},
	}

	ingress.Spec.DefaultBackend = &networkingv1.IngressBackend{
		Service: &networkingv1.IngressServiceBackend{
			Name: "nginx-ingress-default-backend",
			Port: networkingv1.ServiceBackendPort{Number: int32(80)},
		},
	}

	// 这一步非常关键！
	// 建立关联后，删除elasticweb资源时就会将deployment也删除掉
	log.Info("set reference")
	if err := controllerutil.SetControllerReference(exportPort, ingress, r.Scheme); err != nil {
		log.Error(err, "SetControllerReference error")
		return nil, err
	}

	ingress = insertIngress(ctx, r, pods, ingress, exportPort, req)

	err := r.Create(ctx, ingress)
	if err != nil {
		return nil, err
	}

	return ingress, nil
}

func createPodServiceIfNotExists(ctx context.Context, r *ExportPortReconciler, exportPort *exportportv1.ExportPort, req ctrl.Request, pod *corev1.Pod) error {
	log := log.FromContext(ctx)

	service := &corev1.Service{}
	req.NamespacedName.Name = pod.Name
	err := r.Get(ctx, req.NamespacedName, service)

	// 如果查询结果没有错误，证明service正常，就不做任何操作
	if err == nil {
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
			Name:      pod.Name,
			Labels: map[string]string{
				req.NamespacedName.String(): pod.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				SERVICE_TAG: pod.Name,
			},
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

func createPodEndpointIfNotExists(ctx context.Context, r *ExportPortReconciler, exportPort *exportportv1.ExportPort, req ctrl.Request, pod *corev1.Pod) error {
	log := log.FromContext(ctx)

	endponint := &corev1.Endpoints{}
	req.NamespacedName.Name = pod.Name
	err := r.Get(ctx, req.NamespacedName, endponint)

	// 如果查询结果没有错误，证明service正常，就不做任何操作
	if err == nil {
		return nil
	}

	// 如果错误不是NotFound，就返回错误
	if !errors.IsNotFound(err) {
		log.Error(err, "query endponint error")
		return err
	}

	// 实例化一个数据结构
	endponint = &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: exportPort.Namespace,
			Name:      pod.Name,
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: pod.Status.PodIP,
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Port: int32(8080),
					},
				},
			},
		},
	}
	// 这一步非常关键！
	// 建立关联后，删除elasticweb资源时就会将deployment也删除掉
	log.Info("set reference")
	if err := controllerutil.SetControllerReference(exportPort, endponint, r.Scheme); err != nil {
		log.Error(err, "SetControllerReference error")
		return err
	}
	// 创建service
	log.Info("start create endponint")
	if err := r.Create(ctx, endponint); err != nil {
		log.Error(err, "create endponint error")
		return err
	}

	log.Info("create endponint success")

	return nil
}

func (r *ExportPortReconciler) getLabelPods(ctx context.Context, label string) (*corev1.PodList, error) {
	log := log.FromContext(ctx)
	pods := &corev1.PodList{}
	labels := &client.MatchingLabels{label: "true"}

	err := r.ApiReader.List(ctx, pods, labels)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info(fmt.Sprintf("label:[%s] pods:%d", label, 0))
			return nil, nil
		}
		return nil, err
	} else {
		log.Info(fmt.Sprintf("label:[%s] pods:%d", label, len(pods.Items)))
		return pods, nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExportPortReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ApiReader = mgr.GetAPIReader()
	return ctrl.NewControllerManagedBy(mgr).
		For(&exportportv1.ExportPort{}).
		Watches(&source.Kind{Type: &corev1.Pod{}},
			&EnqueueRequestForObject{}). //builder.OnlyMetadata
		//WithEventFilter(&ResourceChangedPredicate{r:r}).
		Complete(r)
}

// 完成了pod的处理后，更新最新状态
func updateStatus(ctx context.Context, r *ExportPortReconciler, exportPort *exportportv1.ExportPort, num int32) error {
	log := log.FromContext(ctx)
	// 当pod创建完毕后，当前系统实际的QPS：单个pod的QPS * pod总数
	// 如果该字段还没有初始化，就先做初始化
	if nil == exportPort.Status.CurrNum {
		exportPort.Status.CurrNum = new(int32)
	}

	*(exportPort.Status.CurrNum) = num

	log.Info(fmt.Sprintf("CurrNum[%d]", *(exportPort.Status.CurrNum)))

	if err := r.Update(ctx, exportPort); err != nil {
		log.Error(err, "update instance error")
		return err
	}

	return nil
}
