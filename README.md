Depends - A small, versatile, and thread safe dependency injection library for Go
=================================================================================

[![GoDoc](https://godoc.org/github.com/jsdw/depends?status.svg)](https://godoc.org/github.com/jsdw/depends)

The purpose of this library is to provide an easy way to register values or value constructors based on their type, and later inject or get back the corresponding values when asking for those types.

Possible uses include:

- Making things easier to test by using Dependency Injection instead of globals. this gives us the chance to intercept and alter values being provided to functions at test time.
- Being a type based store.

Here's what using this library looks like:

```go
import (
	"bytes"
	"fmt"
	"github.com/jsdw/depends"
)

type Foo int
type Bar struct{ value int }
type Wibble string
type ComplicatedType struct{ sum int }

func main() {

	// Create a new context (or use the global one):
	ctx := New()

	// Register some unique types against that context:
	ctx.Register(Foo(100))
	ctx.Register(Bar{100})
	ctx.Register(Wibble("not-wibbly-enough"))

	// We can also register functions, which are called
	// the first time the resulting value is asked for.
	// This can inject other registered things:
	ctx.Register(func(f Foo, b Bar) ComplicatedType {
		return ComplicatedType{int(f) + b.value}
	})

	// Now, we can inject any of the registered items into
	// a function by asking for their corresponding types.
	ctx.Inject(func(foo Foo, bar Bar, c ComplicatedType) {
		fmt.Printf("Foo is %d\n", foo)
		fmt.Printf("Bar is %d\n", bar.value)
		fmt.Printf("ComplicatedType is %d\n", c.sum)
	})

	// Pointers are handled, and allow changing of an injected thing.
	// most of the time you probably won't want to do this.
	ctx.Inject(func(wibble *Wibble) {
		*wibble = Wibble("wibble")
	})

	// Inject calls are blocking, so you can use them to pluck things
	// out of a Context and make available elsewhere.
	var w string
	ctx.Inject(func(wibble Wibble) {
		w = string(wibble)
	})
	fmt.Println(w)

	// Output:
	// Foo is 100
	// Bar is 100
	// ComplicatedType is 200
	// wibble
}
```

For more control over which dependencies are available, we can create our own non-global `Context`s. We can also create child `Context`s which can be used to selectively override or add additional dependencies on top of the parent `Context`, while leaving the parent unchanged:

```go
import (
    "github.com/jsdw/depends"
)

type Foo int
type Bar int

func main() {

    ctx := depends.New()

    ctx.Register(Foo(100))
    ctx.Register(Bar(10))

    childCtx := ctx.Child()

    // override our Foo dependency in the child:
    childCtx.Register(Foo(200))

    ctx.Inject(func(f Foo) {
        // f == Foo(100)
    })

    childCtx.Inject(func(f Foo, b Bar) {
        // f == Foo(200)
        // b == Bar(10)
    })

}
```

## Warning

A word of warning: using this library trades a certain amount of compile time safety (accessing global variables for instance) for run time checks (checking that a dependency has actually been registered when we ask for it). This is an important tradeoff to consider.

In cases wherein dependencies are injected and used early on, a fast failure will be easy to spot, and the advantage of being able to mock, adjust and easily access dependencies can outweigh the downsides. On the other hand, rarely-run functions making use of more obscure dependencies (that you could have forgotten to actually register) could lead to annoying and unnecesary failures.