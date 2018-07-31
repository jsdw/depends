package depends

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

func Example() {

	type Foo int
	type Bar struct{ value int }
	type Wibble string
	type ComplicatedType struct{ sum int }

	// Create a new context:
	ctx := New()

	// Register some unique types against that context:
	ctx.Register(Foo(100))
	ctx.Register(Bar{100})
	ctx.Register(Wibble("not-wibbly-enough"))

	// We can also register functions, which are called
	// the first time the resulting value is asked for.
	// This can also inject other registered things:
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

func ExampleContext_Child() {

	type Foo string
	type Bar string

	ctx := New()
	ctx.Register(Foo("parentFoo"))
	ctx.Register(Bar("parentBar"))

	childCtx := ctx.Child()
	childCtx.Register(Foo("childFoo"))

	ctx.Inject(func(f Foo, b Bar) {
		// prints parentFoo, then parentBar:
		fmt.Println(f)
		fmt.Println(b)
	})

	childCtx.Inject(func(f Foo, b Bar) {
		// prints childFoo, then parentBar:
		fmt.Println(f)
		fmt.Println(b)
	})

	// Output:
	// parentFoo
	// parentBar
	// childFoo
	// parentBar

}

func ExampleContext_Register() {

	type Foo int
	type Bar struct{}

	ctx := New()

	// Register the type Foo:
	ctx.Register(Foo(100))

	// Any attempts to register Foo again will
	// overwrite the previous. Also note that
	// We can Register multiple things at once:
	ctx.Register(Foo(200), Bar{})

	// This includes using pointers. Register
	// will automatically handle any levels of
	// pointer indirection for you:
	foo := Foo(300)
	ctx.Register(&foo)

	// Once a type is registered, it can be asked
	// for by requesting the same type during Inject:
	ctx.Inject(func(f Foo) {

		// This will print "300":
		fmt.Printf("%d", f)

	})

	// Output: 300
}

func ExampleContext_Inject() {

	type Foo int
	type Bar struct{ Value int }
	type Unknown int

	ctx := New()
	ctx.Register(Foo(100), Bar{})

	// You can ask for things by pointer or value:
	ctx.Inject(func(f Foo, fptr *Foo, b Bar) {

	})

	// A panic will occur if you ask for a type which
	// has not been registered (in this case, 'Unknown').
	// use TryInject if you'd like to handle this error
	// gracefully:
	ctx.Inject(func(u Unknown) {
		// never runs
	})

}

func ExampleContext_Inject_interfaces() {

	// We need to surround interfaces in
	// concrete types:
	type R struct{ I io.Reader }
	type W struct{ I io.Writer }

	ctx := New()

	out := bytes.Buffer{}

	// Satisfy our interfaces by instantiating and
	// registering some concrete types:
	ctx.Register(R{strings.NewReader("Read from this")})
	ctx.Register(W{&out})

	// Make use of the interfaces. To test the copyOut
	// function, we could instead inject into it using a
	// context that mocks out the interfaces.
	copyOut := func(r R, w W) {
		io.Copy(w.I, r.I)
	}

	ctx.Inject(copyOut)
	fmt.Println(string(out.Bytes()))

	// Output: Read from this
}

// If we register a function that returns some type, rather than just a type,
// it will be called the first time that we try to inject that type into something.
// The function can itself ask for things to be injected into it.
//
// A panic (or error if TryInject was used) will occur if there is a dependency
// loop; for example, if there is some A which injects some B which injects A.
func ExampleContext_Inject_injection() {

	type Foo struct{ inner int }
	type Bar int
	type Wibble int
	type Zoo int

	ctx := New()

	// When we try to inject Foo for the first time, we ask for the values
	// of Bar and Zoo to be provided and set Foo to be the sum of them:
	ctx.Register(func(h Bar, z Zoo) Foo {
		return Foo{int(h) + int(z)}
	})

	// When we try to inject Bar for the first time, we get the value of Wibble
	// and set Bar to be equal to it:
	ctx.Register(func(w Wibble) Bar {
		return Bar(w)
	})

	ctx.Register(Wibble(2000))
	ctx.Register(Zoo(2))

	ctx.Inject(func(f Foo) {
		// The Foo function sets Foo.inner = Bar + Zoo
		// The Bar function sets Bar = Wibble
		// Thus, this will print "Foo is 2002"
		fmt.Printf("Foo is %d", f.inner)
	})

	// Output: Foo is 2002
}

func ExampleContext_TryInject() {

	type Foo int
	type Unknown int

	ctx := New()
	ctx.Register(Foo(100))

	// This will not run, but instead return a non-nil
	// error as Unknown has not been registered but
	// is being asked for:
	err := ctx.TryInject(func(f Foo, u Unknown) {

	})

	// If we like, we can match on the error type to
	// find out the specific issue (but printing it will
	// return something descriptive):
	switch err.(type) {
	case ErrorFunctionNotProvided:
		fmt.Println("Function not given to TryInject call")
	case ErrorCircularInject:
		fmt.Println("Registration function loop")
	case ErrorTypeNotRegistered:
		fmt.Println("Type not registered")
	}

	// Output: Type not registered
}
