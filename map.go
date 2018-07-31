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
	// If not nil, this is a function that can have
	// dependencies injected into, and will be called
	// in order to return the desired thing.
	itemMaker func(from []reflect.Type) (reflect.Value, error)
	// If not zero, this is the item (either provided
	// directly or once it's returned from the itemMaker)
	item reflect.Value
	// Make sure that the item init only happens once.
	// This will run itemMaker if not nil and populate
	// item.
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
