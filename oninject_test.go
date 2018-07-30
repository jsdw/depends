package depends

import "fmt"

// If we attach an OnInjection method to the pointer receiver of a type,
// it will be called the first time that we try to inject that type into something.
// The method can itself ask for things to be injected into it.
//
// A panic (or error if TryInject was used) will occur if there is a dependency
// loop; for example, if there is some A which injects some B which injects A.
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

func ExampleContext_Inject_injection() {

	ctx := New()

	ctx.Register(Foo{})
	ctx.Register(Bar(100))
	ctx.Register(Wibble(2000))
	ctx.Register(Zoo(2))

	ctx.Inject(func(f Foo) {
		// Bar.OnInjection sets Bar = Wibble
		// Foo.OnInjection sets Foo.inner = Bar + Zoo
		// Thus, this will print "Foo is 2002"
		fmt.Printf("Foo is %d", f.inner)
	})

	// Output: Foo is 2002
}
