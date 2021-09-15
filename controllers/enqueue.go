/*
Copyright 2018 The Kubernetes Authors.

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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
)

var enqueueLog = log.FromContext(context.Background()).WithName("eventhandler").WithName("EnqueueRequestForObject")

// EnqueueRequestForObject enqueues a Request containing the Name and Namespace of the object that is the source of the Event.
// (e.g. the created / deleted / updated objects Name and Namespace).  handler.EnqueueRequestForObject is used by almost all
// Controllers that have associated Resources (e.g. CRDs) to reconcile the associated Resource.
type EnqueueRequestForObject struct{}

func (e *EnqueueRequestForObject) AddQueue(key string, q workqueue.RateLimitingInterface) {
	instance := DefaultLabelMap.Get(key)
	if instance != nil {
		q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Name:      instance.GetName(),
			Namespace: instance.GetNamespace(),
		}})
	}
}

// Create implements EventHandler
func (e *EnqueueRequestForObject) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		enqueueLog.Error(nil, "CreateEvent received with no metadata", "event", evt)
		return
	}

	enqueueLog.Info(fmt.Sprintf("[kind:%s]", evt.Object.GetObjectKind().GroupVersionKind().String()))

	if p, ok := evt.Object.(*corev1.Pod); ok {
		enqueueLog.Info(fmt.Sprintf("kindPod[%s]", p.Name))

		if k := DefaultLabelMap.MatchLabels(p.GetLabels()); k != "" {
			e.AddQueue(k, q)
		} else {
			if strings.Contains(p.Name, "tomcat") {
				//enqueueLog.Error(nil, "CreateEvent received pod no label", "event", evt)
			}
		}
		return
	}

	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      evt.Object.GetName(),
		Namespace: evt.Object.GetNamespace(),
	}})
}

// Update implements EventHandler
func (e *EnqueueRequestForObject) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if evt.ObjectOld != nil {
		if p, ok := evt.ObjectOld.(*corev1.Pod); ok {
			if k := DefaultLabelMap.MatchLabels(p.GetLabels()); k != "" {
				e.AddQueue(k, q)
			} else {
				//enqueueLog.Error(nil, "UpdateEvent received pod no label", "event", evt)
			}
			return
		}

		q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Name:      evt.ObjectOld.GetName(),
			Namespace: evt.ObjectOld.GetNamespace(),
		}})
	} else {
		enqueueLog.Error(nil, "UpdateEvent received with no old metadata", "event", evt)
	}

	if evt.ObjectNew != nil {
		if p, ok := evt.ObjectNew.(*corev1.Pod); ok {
			if k := DefaultLabelMap.MatchLabels(p.GetLabels()); k != "" {
				e.AddQueue(k, q)
			} else {
				//enqueueLog.Error(nil, "UpdateEvent received pod no label", "event", evt)
			}
			return
		}

		q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Name:      evt.ObjectNew.GetName(),
			Namespace: evt.ObjectNew.GetNamespace(),
		}})
	} else {
		enqueueLog.Error(nil, "UpdateEvent received with no new metadata", "event", evt)
	}
}

// Delete implements EventHandler
func (e *EnqueueRequestForObject) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		enqueueLog.Error(nil, "DeleteEvent received with no metadata", "event", evt)
		return
	}

	if p, ok := evt.Object.(*corev1.Pod); ok {
		if k := DefaultLabelMap.MatchLabels(p.GetLabels()); k != "" {
			e.AddQueue(k, q)
		} else {
			//enqueueLog.Error(nil, "GenericEvent received pod no label", "event", evt)
		}
		return
	}

	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      evt.Object.GetName(),
		Namespace: evt.Object.GetNamespace(),
	}})
}

// Generic implements EventHandler
func (e *EnqueueRequestForObject) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		enqueueLog.Error(nil, "GenericEvent received with no metadata", "event", evt)
		return
	}

	if p, ok := evt.Object.(*corev1.Pod); ok {
		if k := DefaultLabelMap.MatchLabels(p.GetLabels()); k != "" {
			e.AddQueue(k, q)
		} else {
			//enqueueLog.Error(nil, "GenericEvent received pod no label", "event", evt)
		}
		return
	}

	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      evt.Object.GetName(),
		Namespace: evt.Object.GetNamespace(),
	}})
}
