package depends

import (
	"fmt"
	"reflect"
)

// ErrorFunctionNotProvided is returned from TryInject when
// the argument passed to it is not a function
type ErrorFunctionNotProvided struct{}

func (t ErrorFunctionNotProvided) Error() string {
	return "Inject/TryInject require a function to be provided"
}

// ErrorTypeNotRegistered is returned from TryInject when the
// type asked to be injected has not been registered yet
type ErrorTypeNotRegistered struct {
	// The type that was not found
	Ty reflect.Type
	// The position (1 indexed) of the argument in the function
	// that was handed to TryInject
	Pos int
}

func (t ErrorTypeNotRegistered) Error() string {
	return fmt.Sprintf("Injection of argument %d failed since the type '%s' has not been registered", t.Pos, typeName(t.Ty))
}

// ErrorCircularInject is returned from TryInject when there is a
// circular injection loop
type ErrorCircularInject struct {
	// A slice of the types encountered in the order that
	// OnInjection was called on them
	Chain []reflect.Type
}

func (t ErrorCircularInject) Error() string {
	s := "Injection cycle: "
	for i, ty := range t.Chain {
		s += typeName(ty)
		if i < len(t.Chain)-1 {
			s += " -> "
		}
	}
	return s
}
