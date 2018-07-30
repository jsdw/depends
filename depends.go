// Package depends is a small, versatile dependency injection library.
//
// The basic usage is that you create a Context to register different
// types again, and then you ask for things by type using the Inject or
// TryInject function.
//
// A global Context is provided by default for convenience.
//
// The main use case envisioned for this package is as an alternative for
// using global values. By using a Context instead, it is possible to
// provide mock or alternate values during testing or when certain code
// demands it.
//
// Check out the examples and tests to find out more about how this
// package can be used
//
package depends

import (
	"reflect"
	"sync"
)

// Context is the owner of dependencies. A global context is available for convenience,
// or one can create their own.
type Context struct {
	parent      *Context
	injectables syncMap
}

// This method name is searched for when a thing is first being injected. If it exists, it is
// called, and anything asked for by it is injected into it.
const injectMethodName = "OnInjection"

// New creates a new Context
func New() *Context {
	return &Context{
		parent:      nil,
		injectables: syncMap{},
	}
}

// Child creates a child context. This Context can use anything registered
// with it's parent, but the inverse is not true: anything registered on it
// will not be visible to the parent context.
func (ctx *Context) Child() *Context {
	childCtx := New()
	childCtx.parent = ctx
	return childCtx
}

// Register registers a dependency into the Context to later be asked for
// with Inject or TryInject.
func (ctx *Context) Register(items ...interface{}) {
	for _, item := range items {
		ctx.registerOne(item)
	}
}

func (ctx *Context) registerOne(item interface{}) {
	val := reflect.ValueOf(item)
	ty := val.Type()
	ctx.injectables.put(normalizeKey(ty), &injectableValue{
		item: normalizeValue(val),
		init: sync.Once{},
	})
}

// Inject injects the dependencies asked for into the function provided. If anything
// goes wrong, it will panic. It's expected that this will be used in favour of TryInject
// in most cases, since failure to inject something is normally a sign of programmer error.
//
// If a type that is being injected has an OnInjection method attached to it, that method
// will be run once just before the first attempt to inject the type. Arguments to this
// method are themselves injected into it. This allows for lazy initialisation and
// initialisation that depends on other injected types. See the Injection example.
func (ctx *Context) Inject(fn interface{}) {
	err := ctx.TryInject(fn)

	if err != nil {
		panic(err.Error())
	}
}

// TryInject injects the dependencies asked for into the function provided. If anything
// goes wrong, the function provided is not called and instead an error is returned
// describing the issue.
func (ctx *Context) TryInject(fn interface{}) error {
	fnVal := reflect.ValueOf(fn)
	return ctx.injectIntoFunction(nil, nil, fnVal)
}

func (ctx *Context) injectIntoFunction(from []reflect.Type, fnRecv *reflect.Value, fnVal reflect.Value) error {
	fnTy := fnVal.Type()
	if fnTy.Kind() != reflect.Func {
		return ErrorFunctionNotProvided{}
	}

	argCount := fnTy.NumIn()
	args := []reflect.Value{}

	// if a receiver is given, append that as arg one:
	if fnRecv != nil {
		args = append(args, *fnRecv)
	}

	// start after the receiver type if one given, else look at
	// type of all function args and inject them:
	for i := len(args); i < argCount; i++ {
		argTy := fnTy.In(i)
		argVal, err := ctx.getInjectable(from, argTy)
		if err != nil {
			switch e := err.(type) {
			// We need to add extra info to this error:
			case ErrorTypeNotRegistered:
				e.Pos = i + 1
				return e
			default:
				return err
			}
		}
		args = append(args, argVal)
	}

	_ = fnVal.Call(args)
	return nil
}

func (ctx *Context) getInjectable(from []reflect.Type, ty reflect.Type) (reflect.Value, error) {
	normalKey := normalizeKey(ty)
	arg, ok := ctx.injectables.get(normalKey)
	normalTy := normalKey.Ty

	// Delegate to a parent Context if one exists, else error:
	if !ok {
		if ctx.parent != nil {
			return ctx.parent.getInjectable(from, ty)
		} else {
			return reflect.Value{}, ErrorTypeNotRegistered{Ty: normalTy}
		}
	}

	// run the instantiation method for an injected thing exactly once if one is present.
	// the Injectable interface needs to be given a pointer receiver in order to match this.
	var initErr error
	arg.init.Do(func() {

		valPtr := arg.item
		valPtrTy := valPtr.Type()

		if method, hasMethod := valPtrTy.MethodByName(injectMethodName); hasMethod {

			// if the type we key on has already been seen, complain as we've hit a loop:
			if typeExistsInSlice(from, normalTy) {
				initErr = ErrorCircularInject{appendType(from, normalTy)}
				return
			}

			nextFrom := appendType(from, normalTy)
			injectErr := ctx.injectIntoFunction(nextFrom, &valPtr, method.Func)
			if injectErr != nil {
				initErr = injectErr
				return
			}

		}

	})
	if initErr != nil {
		return reflect.Value{}, initErr
	}

	return denormalizeValue(arg.item, ty)
}
