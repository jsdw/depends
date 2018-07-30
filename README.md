Depends - A small, versatile dependency injection library for Go
================================================================

[![GoDoc](https://godoc.org/github.com/jsdw/depends/web?status.svg)](https://godoc.org/github.com/jsdw/depends/web)

The purpose of this library is to provide an easy way to register and inject
dependencies into functions.

One use case of this approach is to replace global variables in many places.
The advantage of doing so is that you can override them when needed by creating
a child Context, or mock them during testing.

```
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
