package depends

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func Example() {

	type Foo int
	type Bar struct{ value int }
	type Wibble string

	// Create a new context:
	ctx := New()

	// Register some unique types against that context:
	ctx.Register(Foo(100))
	ctx.Register(Bar{100})
	ctx.Register(Wibble("not-wibbly-enough"))

	// Now, we can inject any of those items into a function
	// by asking for their corresponding type:
	ctx.Inject(func(foo Foo, bar Bar) {
		if int(foo) != 100 || int(foo) != bar.value {
			panic("expected foo and bar to be 100")
		}
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

	// Output: wibble
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

	// Prints "Read from this"
	fmt.Println(string(out.Bytes()))

	// Output: Read from this
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
		fmt.Println("OnInjection loop")
	case ErrorTypeNotRegistered:
		fmt.Println("Type not registered")
	}

	// Output: Type not registered
}

// Context should sort itself out if not called with New,
// just in case
func TestLazyInit(t *testing.T) {

	c := Context{}

	c.Register(12)
	c.Inject(func(n int) {})
}

// We can inject basic things by their types
func TestBasicInjection(t *testing.T) {

	type First int
	type Second int

	ctx := New()
	ctx.Register(First(12))
	ctx.Register(Second(100))

	// empty func should never have an issue:
	ctx.Inject(func() {})

	err := ctx.TryInject(func(h First) {
		if h != First(12) {
			t.Error("argument is not what we expected (1)")
		}
	})
	if err != nil {
		t.Error("Injecting 'Hello' failed but should have been successful (1)")
	}

	err = ctx.TryInject(func(h First, s Second) {
		if h != First(12) || s != Second(100) {
			t.Error("arguments are not what we expected (2)")
		}
	})
	if err != nil {
		t.Error("Injecting 'Hello' failed but should have been successful (2)")
	}

	err = ctx.TryInject(func(h First, h2 First, h3 First) {
		if h != First(12) || h2 != h || h3 != h {
			t.Error("arguments are not what we expected (3)")
		}
	})
	if err != nil {
		t.Error("Injecting 'Hello' failed but should have been successful (3)")
	}

	err = ctx.TryInject(func(h First, h2 *First, h3 **First) {
		if h != First(12) || *h2 != h || **h3 != h {
			t.Error("arguments are not what we expected (4)")
		}
	})
	if err != nil {
		t.Error("Injecting 'Hello' failed but should have been successful (4)")
	}

}

// We'll get an error/panic if we try injecting something
// that the injector does not know about.
func TestFailedInjection(t *testing.T) {

	type First int
	type Second int
	type Unknown int

	ctx := New()
	ctx.Register(First(12))
	ctx.Register(Second(100))

	err := ctx.TryInject(func(u Unknown) {

	})
	if err == nil {
		t.Error("Injecting 'Unknown' should have failed but did not (1)")
	}

	err = ctx.TryInject(func(f First, u Unknown) {

	})
	if err == nil {
		t.Error("Injecting 'Unknown' should have failed but did not (2)")
	}

}

// We can inject interfaces as well, but we have to wrap them in
// concrete types owing to how reflection works. Using interfaces
// in this way allows us to replace things with mocks to test.
type Thinger interface {
	GetThings() int
}

type ThingerContainer struct {
	Interface Thinger
}

type Thing int

func (t Thing) GetThings() int {
	return int(t)
}

func TestInterfaceInjection(t *testing.T) {

	ctx := New()

	// interfaces themselves can't be passed straight
	// into the registration unfortunately, because the
	// interface details are not preserved. Thus, if we want
	// to inject an interface, we need to wrap it into a
	// concrete type like so:
	ctx.Register(ThingerContainer{Thing(100)})

	err := ctx.TryInject(func(thinger ThingerContainer) {
		val := thinger.Interface.GetThings()
		if val != 100 {
			t.Error("argument is not what we expected")
		}
	})
	if err != nil {
		t.Errorf("Injecting 'Thinger' failed but should have been successful: %s", err)
	}

}

// We can ask for pointers or not-pointers to things; both should
// work and return the same injected value:
func TestPointersAndNonPointers(t *testing.T) {

	type Thing int

	ctx := New()
	th := Thing(100)
	ctx.Register(&th)

	ctx.Inject(func(t1 Thing, t2 *Thing, t3 **Thing) {
		if t1 != Thing(100) || *t2 != t1 || **t3 != t1 {
			t.Error("Mismatched values")
		}
	})

}

// We can alter injected things by asking for them by pointer:
func TestPointersCanAlterInjected(t *testing.T) {

	ctx := New()

	type Thing int

	ctx.Register(Thing(100))

	ctx.Inject(func(tp *Thing) {
		*tp = Thing(200)
	})

	ctx.Inject(func(tp Thing) {
		if tp != Thing(200) {
			t.Error("Thing did not change")
		}
	})

	ctx.Inject(func(tp ***Thing) {
		***tp = Thing(300)
	})

	ctx.Inject(func(tp Thing) {
		if tp != Thing(300) {
			t.Error("Thing did not change again")
		}
	})

}

// We can pass pointers when registering and inject with any number
// of pointers (well, any reasonable number).
func TestPointerInjection(t *testing.T) {

	type Foo int
	type Bar int
	type Wibble int

	ctx := New()

	ctx.Register(Foo(1))
	b := Bar(1)
	ctx.Register(&b)
	w := Wibble(1)
	wp := &w
	ctx.Register(&wp)

	ctx.Inject(func(f Foo, f1 *Foo, f2 **Foo, b Bar, b1 *Bar, b2 **Bar, w Wibble) {

	})

}

// We can define an interface on the type to be injected which can itself
// ask for injected things and runs on first attempt to ask for the injected
// item.
type TestInject struct {
	inner int
}

func (ti *TestInject) OnInjection(h TestInject2) {
	ti.inner = int(h)
}

type TestInject2 int

func (ti *TestInject2) OnInjection(h TestInject3) {
	*ti = TestInject2(h)
}

type TestInject3 int

//// uncomment this to cause a dependency cycle:
// func (ti *TestInject3) OnInjection(h TestInject) {
// 	*ti = TestInject3(h.inner)
// }

func TestOnInject(t *testing.T) {

	ctx := New()

	ctx.Register(TestInject{})
	ctx.Register(TestInject2(100))
	ctx.Register(TestInject3(2000))

	ctx.Inject(func(i TestInject) {
		if i.inner != 2000 {
			t.Error("OnInjection failed to set up value")
		}
	})

}

// The OnInjection function should be called only once
// (and synchronous Inject should be OK)
type OnInjectOnce struct {
	times int32
}

func (t *OnInjectOnce) OnInjection() {
	atomic.AddInt32(&t.times, 1)
}

func TestOnInjectCalledOnce(t *testing.T) {

	ctx := New()

	ctx.Register(OnInjectOnce{times: 0})

	wg := sync.WaitGroup{}
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go ctx.Inject(func(th OnInjectOnce) {
			wg.Done()
		})
	}
	wg.Wait()

	ctx.Inject(func(th OnInjectOnce) {
		if th.times != 1 {
			t.Error("init called more than once")
		}
	})

}

// The child context can see anything a parent can, but not the
// other way around
func TestChildContext(t *testing.T) {

	type Foo string
	type Bar string

	ctx := New()
	ctx.Register(Foo("parentFoo"))
	ctx.Register(Bar("parentBar"))

	childCtx := ctx.Child()
	childCtx.Register(Foo("childFoo"))

	ctx.Inject(func(f Foo, b Bar) {
		if f != Foo("parentFoo") || b != Bar("parentBar") {
			t.Error("child context modified parent")
		}
	})

	childCtx.Inject(func(f Foo, b Bar) {
		if f != Foo("childFoo") || b != Bar("parentBar") {
			t.Error("child context could not see own overrides")
		}
	})
}
