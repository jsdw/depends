package depends

import (
	"reflect"
	"sync"
)

type syncMap struct {
	Store sync.Map
}

type injectableKey struct {
	Ty reflect.Type
}

type injectableValue struct {
	item reflect.Value
	init sync.Once
}

func (m *syncMap) get(key injectableKey) (*injectableValue, bool) {
	val, ok := m.Store.Load(key)
	if !ok {
		return &injectableValue{}, false
	}
	return val.(*injectableValue), ok
}

func (m *syncMap) put(key injectableKey, val *injectableValue) {
	m.Store.Store(key, val)
}
