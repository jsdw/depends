package depends

import (
	"sync"
	"testing"
)

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

// Test that we can tweak the original pointer to a thing we
// injected if we want.
func TestOriginalPointerStillUseful(t *testing.T) {

	type Thing struct {
		Val int
	}
	type NotPointedTo struct {
		Val int
	}

	thing := &Thing{100}
	notPointedTo := NotPointedTo{100}

	ctx := New()
	ctx.Register(thing)
	ctx.Register(notPointedTo)

	ctx.Inject(func(thing Thing, np NotPointedTo) {
		if thing.Val != 100 {
			t.Error("thing val should start at 100")
		}
		if np.Val != 100 {
			t.Error("np val should start at 100")
		}
	})

	thing.Val = 200
	notPointedTo.Val = 200

	ctx.Inject(func(thing Thing, np NotPointedTo) {
		if thing.Val != 200 {
			t.Error("VAl should now be 200")
		}
		if np.Val != 100 {
			t.Error("np val should still be 100")
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

// We can register functions that return the things we're interested
// in if we want to do some initialisation etc before the thing is
// first obtained. These functions are called exactly once
type TestInject struct{ inner int }
type TestInject2 int
type TestInject3 int

func TestRegisterFactories(t *testing.T) {

	ctx := New()

	ctx.Register(func(h TestInject2) TestInject {
		return TestInject{int(h)}
	})
	ctx.Register(func(h TestInject3) TestInject2 {
		return TestInject2(int(h))
	})
	ctx.Register(TestInject3(2000))

	ctx.Inject(func(i TestInject) {
		if i.inner != 2000 {
			t.Error("registration functions failed to set up value")
		}
	})

}

// The function provided to Register should be called only once
// (and synchronous Inject should be OK)
func TestRegisterFactoryCalledOnce(t *testing.T) {

	type Foo struct{}

	ctx := New()

	times := 0

	ctx.Register(func() Foo {
		times++
		return Foo{}
	})

	wg := sync.WaitGroup{}
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go ctx.Inject(func(th Foo) {
			wg.Done()
		})
	}
	wg.Wait()

	ctx.Inject(func(th Foo) {
		if times != 1 {
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
