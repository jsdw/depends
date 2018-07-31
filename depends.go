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
// A word of warning: using this library trades a certain amount of
// compile time safety (accessing global variables for instance) for run
// time checks (checking that a dependency has actually been registered
// when we ask for it). This is an important tradeoff to consider.
//
// In cases wherein dependencies are injected and used early on, a fast
// failure will be easy to spot, and the advantage of being able to mock,
// adjust and easily access dependencies can outweigh the downsides. On
// the other hand, rarely-run functions making use of more obscure
// dependencies (that you could have forgotten to actually register)
// could lead to annoying and unnecesary failures.
//
package depends

import (
	"fmt"
	"reflect"
)

// Context is the owner of dependencies. A global context is available for convenience,
// or one can create their own.
type Context struct {
	parent      *Context
	injectables syncMap
}

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
// with Inject or TryInject. We can register some type itself, or alternately
// we can register a function that returns some type.
//
// In the latter case, the function will be run the first time the type is
// asked for. Anything the function asks for as an argument will be injected
// into it, allowing for complex dependencies between registered types.
func (ctx *Context) Register(items ...interface{}) {
	for _, item := range items {
		ctx.registerOne(item)
	}
}

func (ctx *Context) registerOne(item interface{}) {
	val := reflect.ValueOf(item)
	ty := val.Type()
	kind := ty.Kind()

	if kind == reflect.Func {

		if ty.NumOut() != 1 {
			panic(fmt.Sprintf(
				"If registering a function, it must return exactly one value"+
					"of the type you'd like to be able to Inject, but the function"+
					"provided returns %d items", ty.NumOut()))
		}

		outTy := ty.Out(0)
		ctx.injectables.put(normalizeKey(outTy), &injectableValue{
			itemMaker: func(from []reflect.Type) (reflect.Value, error) {
				vals, err := ctx.injectIntoFunction(from, nil, val)
				if err != nil {
					return reflect.Value{}, err
				}
				return normalizeValue(vals[0]), nil
			},
		})

	} else {

		ctx.injectables.put(normalizeKey(ty), &injectableValue{
			item: normalizeValue(val),
		})

	}

}

// Inject injects the dependencies asked for into the function provided. If anything
// goes wrong, it will panic. It's expected that this will be used in favour of TryInject
// in most cases, since failure to inject something is normally a sign of programmer error.
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
	_, err := ctx.injectIntoFunction(nil, nil, fnVal)
	return err
}

func (ctx *Context) injectIntoFunction(from []reflect.Type, fnRecv *reflect.Value, fnVal reflect.Value) (out []reflect.Value, outErr error) {
	fnTy := fnVal.Type()
	if fnTy.Kind() != reflect.Func {
		return []reflect.Value{}, ErrorFunctionNotProvided{}
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
				outErr = e
				return
			default:
				outErr = e
				return
			}
		}
		args = append(args, argVal)
	}

	// recover from any panic that occurs when calling the function:
	defer func() {
		if e := recover(); e != nil {
			outErr = ErrorPanicInFunction{e}
		}
	}()

	out = fnVal.Call(args)
	return
}

func (ctx *Context) getInjectable(from []reflect.Type, ty reflect.Type) (reflect.Value, error) {
	normalKey := normalizeKey(ty)
	arg, ok := ctx.injectables.get(normalKey)
	normalTy := normalKey.Ty

	// Delegate to a parent Context if one exists, else error:
	if !ok {
		if ctx.parent != nil {
			return ctx.parent.getInjectable(from, ty)
		}
		return reflect.Value{}, ErrorTypeNotRegistered{Ty: normalTy}
	}

	// run the instantiation method for an injected thing exactly once if one is present.
	// the Injectable interface needs to be given a pointer receiver in order to match this.
	var initErr error
	arg.init.Do(func() {

		// if we have an itemMaker we need to run it to get our item, otherwise bail.
		if arg.itemMaker == nil {
			return
		}

		// if the type we key on has already been seen, complain as we've hit a loop:
		if typeExistsInSlice(from, normalTy) {
			initErr = ErrorCircularInject{appendType(from, normalTy)}
			return
		}

		// run the item maker to create our item, passing our chain of seen types.
		res, err := arg.itemMaker(from)
		initErr = err
		arg.item = res

	})
	if initErr != nil {
		return reflect.Value{}, initErr
	}

	return denormalizeValue(arg.item, ty)
}
