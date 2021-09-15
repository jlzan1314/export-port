package controllers

import (
	exportportv1 "export-port/api/v1"
	"sync"
)

type LabelMap struct {
	sync.Map
}

var DefaultLabelMap = &LabelMap{}

func (m *LabelMap) MatchLabels(l map[string]string) string {
	result := ""
	m.Range(func(key, value interface{}) bool {
		s, _ := key.(string)
		if a, ok := l[s]; ok && a == "true" {
			result = s
			return false
		}
		return true
	})
	return result
}

func (m *LabelMap) Get(key string) *exportportv1.ExportPort {
	i, ok := m.Load(key)
	if !ok {
		return nil
	}
	d, ok := i.(*exportportv1.ExportPort)
	if !ok {
		return nil
	}
	return d
}
