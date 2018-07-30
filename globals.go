package depends

// A global context is provided for convenience:
var context = New()

// Child creates a child context. This Context can use anything registered
// with the global Context, but the inverse is not true: anything registered on it
// will not be visible to the global context.
func Child() *Context {
	return context.Child()
}

// Register registers a dependency into a global Context to later be used
func Register(items ...interface{}) {
	context.Register(items...)
}

// TryInject injects the dependencies asked for from the global context into the
// function provided. If anything goes wrong, the function provided is not called
// and instead an error is returned describing the issue.
func TryInject(fn interface{}) error {
	return context.TryInject(fn)
}

// Inject injects the dependencies asked for into the function provided. If anything
// goes wrong, it will panic. It's expected that this will be used in favour of TryInject
// in most cases, since failure to inject something is normally a sign of programmer error.
func Inject(fn interface{}) {
	context.Inject(fn)
}
