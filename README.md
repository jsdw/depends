Depends - A small, versatile, and thread safe dependency injection library for Go
=================================================================================

[![GoDoc](https://godoc.org/github.com/jsdw/depends?status.svg)](https://godoc.org/github.com/jsdw/depends)

The purpose of this library is to provide an easy way to register and inject dependencies into functions.

One use case of this approach is to replace global variables in many places. The advantage of doing so is that you can override them when needed by creating a child Context, or mock them during testing.

```go
import (
	"bytes"
	"fmt"
	"github.com/jsdw/depends"
)

type Foo int
type Bar struct{ value int }
type Wibble string

func main() {
	// register some things we might otherwise expose as globals:
	depends.Register(Foo(100))
	depends.Register(Bar{100})
	depends.Register(Wibble("not-wibbly-enough"))

	// now, we can inject registered items into a function
	// by asking for their corresponding type:
	depends.Inject(func(foo Foo, bar Bar) {
		if int(foo) != 100 || int(foo) != bar.value {
			panic("expected foo and bar to be 100")
		}
	})

	// pointers are handled, and allow changing of an injected thing.
	// most of the time you probably won't want to do this.
	depends.Inject(func(wibble *Wibble) {
		*wibble = Wibble("wibble")
	})

	// Inject calls are blocking, so you can use them to pluck things
	// out of a Context and make available elsewhere.
	var w string
	depends.Inject(func(wibble Wibble) {
		w = string(wibble)
	})

    // Prints "wibble":
	fmt.Println(w)
}
```

Types can do some initialisation just prior to the first time that they are injected anywhere by having an `OnInjection` method. Any arguments provided to this function will also be injected through the same context.

This allows for lazy initialisation and initialisation which depends on the values of other injected types:

```go
import (
    "fmt"
    "github.com/jsdw/depends"
)

type Foo struct {
	inner int
}

// When we try to inject Foo for the first time, we get the values of Bar
// and Zoo and set Foo to be the sum of them:
func (ti *Foo) OnInjection(h Bar, z Zoo) {
	ti.inner = int(h) + int(z)
}

type Bar int

// When we try to inject Bar for the first time, we get the value of Wibble
// and set Bar to be equal to it:
func (ti *Bar) OnInjection(h Wibble) {
	*ti = Bar(h)
}

type Wibble int

type Zoo int

func main() {

	depends.Register(Foo{})
	depends.Register(Bar(100))
	depends.Register(Wibble(2000))
	depends.Register(Zoo(2))

	depends.Inject(func(f Foo) {
		// Bar.OnInjection sets Bar = Wibble
		// Foo.OnInjection sets Foo.inner = Bar + Zoo
		// Thus, this will print "Foo is 2002"
		fmt.Printf("Foo is %d", f.inner)
	})

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