package depends

import (
	"fmt"
	"reflect"
)

func typeName(ty reflect.Type) string {
	s := ""
	for {
		if ty.Kind() == reflect.Ptr {
			ty = ty.Elem()
			s += "*"
		} else {
			break
		}
	}
	s += ty.Name()
	return s
}

func appendType(s []reflect.Type, ty reflect.Type) []reflect.Type {
	out := make([]reflect.Type, 0, len(s)+1)
	for _, item := range s {
		out = append(out, item)
	}
	out = append(out, ty)
	return out
}

func typeExistsInSlice(s []reflect.Type, ty reflect.Type) bool {
	for _, sTy := range s {
		if ty == sTy {
			return true
		}
	}
	return false
}

func normalizeKey(ty reflect.Type) injectableKey {
	for {
		if ty.Kind() == reflect.Ptr {
			ty = ty.Elem()
		} else {
			break
		}
	}
	return injectableKey{ty}
}

func normalizeValue(val reflect.Value) reflect.Value {

	if val.Kind() != reflect.Ptr {
		// Add a level of indirection if there isn't one,
		// so that we can provide a pointer back and allow
		// modification of stored value
		ptr := reflect.New(val.Type())
		ptr.Elem().Set(val)
		return ptr
	}

	// We have a pointer. deref until we don't, and then
	// return the last pointer we saw.
	var lastPtr reflect.Value
	for {
		if val.Kind() == reflect.Ptr {
			lastPtr = val
			val = val.Elem()
		} else {
			return lastPtr
		}
	}

}

func denormalizeValue(val reflect.Value, targetType reflect.Type) (reflect.Value, error) {

	// if match, return quick:
	if val.Type() == targetType {
		return val, nil
	}

	// deref and return once match is found:
	dval := val
	for {
		if dval.Kind() == reflect.Ptr {
			dval = dval.Elem()
			if dval.Type() == targetType {
				return dval, nil
			}
		} else {
			break
		}
	}

	// no luck? add more indirections then until match found
	// (up to some sensible limit, incase we will never find the type)
	rval := val
	for i := 0; i < 50; i++ {
		ptr := reflect.New(rval.Type())
		ptr.Elem().Set(rval)
		rval = ptr
		if rval.Type() == targetType {
			return rval, nil
		}
	}

	// something likely went wrong :(
	return reflect.Value{}, fmt.Errorf("failed to denormalize value of type '%s' to expected type '%s'", typeName(val.Type()), typeName(targetType))
}
