package controllers

import (
	"export-port/api/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sync"
)

var m = new(sync.Map)

type ResourceChangedPredicate struct {
	r *ExportPortReconciler
	predicate.Funcs
}

func (rl *ResourceChangedPredicate) Update(e event.UpdateEvent) bool {
	a, ok1 := e.ObjectOld.(*corev1.Pod)
	b, ok2 := e.ObjectNew.(*corev1.Pod)
	if ok1 && ok2 {
		return cmpLabels(a) && cmpLabels(b)
	}
	_, ok1 = e.ObjectOld.(*v1.ExportPort)
	_, ok2 = e.ObjectNew.(*v1.ExportPort)
	if ok1 && ok2 {
		return true
	}
	return false
}

func (rl *ResourceChangedPredicate) Create(e event.CreateEvent) bool {
	p, ok1 := e.Object.(*corev1.Pod)
	if ok1 {
		return cmpLabels(p)
	}

	a, ok1 := e.Object.(*v1.ExportPort)
	if ok1 {
		m.Store(a.Spec.Label, "1")
		return true
	}

	return false
}

func cmpLabels(pod *corev1.Pod) bool {
	l := pod.GetLabels()
	result := false

	m.Range(func(key, value interface{}) bool {
		s, _ := key.(string)
		if _, ok := l[s]; ok {
			result = true
			return false
		}
		return true
	})
	return result
}

func (rl *ResourceChangedPredicate) Delete(e event.DeleteEvent) bool {
	a, ok1 := e.Object.(*corev1.Pod)
	if ok1 {
		return cmpLabels(a)
	}
	_, ok1 = e.Object.(*v1.ExportPort)
	if ok1 {
		return true
	}
	return false
}
