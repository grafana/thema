package thema

import (
	"fmt"
	"reflect"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var linstr = `name: "single"
schemas: [{
	version: [0, 0]
	schema: {
		astring?: string
		anint:   int64 | *42
		abool:   bool
	}
	examples: {
		simple: {
			astring: "some string"
			anint:  42
			abool:  true
		}
	}
}]
`

type TestType struct {
	Astring *string `json:"astring"`
	Anint   int64   `json:"anint"`
	Abool   bool    `json:"abool"`
}

func testLin(linstr string) Lineage {
	rt := NewRuntime(cuecontext.New())
	val := rt.Context().CompileString(linstr)
	lin, err := BindLineage(val, rt)
	if err != nil {
		panic(err)
	}
	return lin
}

func ptr[T any](t T) *T {
	return &t
}

func TestBindType(t *testing.T) {
	lin := testLin(linstr)

	tt := &TestType{Astring: ptr("init"), Anint: 10}
	ts, err := BindType[*TestType](lin.First(), tt)
	if err != nil {
		t.Fatal(errors.Details(err, nil))
	}

	nt1 := ts.NewT()
	if nt1.Astring != nil || nt1.Anint == 10 {
		t.Fatalf("values set on parameter to BindType showed up on NewT(): %v", nt1)
	}
	if nt1.Anint != 42 {
		t.Fatalf("expected schema-specified default of 42 for nt1.Anint, got %v", nt1.Anint)
	}

	// Now, ensure values set on returned type don't leak into next return from NewT()
	nt1.Astring = ptr("nt1")
	nt1.Anint = 10

	nt2 := ts.NewT()
	if nt2.Astring != nil || nt2.Anint == 10 {
		t.Fatalf("values from nt1 leaked into nt2: %v", nt2)
	}
	if nt2.Anint != 42 {
		t.Fatalf("expected schema-specified default of 42 for nt2.Anint, got %v", nt2.Anint)
	}
}

func TestSchema_Examples(t *testing.T) {
	t.Run("withExamples", func(t *testing.T) {
		lin := testLin(linstr)

		sch := lin.First()
		examples := sch.Examples()
		require.NotNil(t, examples)

		// There must be a "simple" example based on
		// the definition above (beginning of file).
		require.NotEmpty(t, examples)
		require.NotNil(t, examples["simple"])

		ts, err := BindType[*TestType](sch, &TestType{})
		require.NoError(t, err)

		tinst, err := ts.ValidateTyped(examples["simple"].Underlying())
		require.NoError(t, err)

		got, err := tinst.Value()
		require.NoError(t, err)

		expected := &TestType{Astring: ptr("some string"), Anint: 42, Abool: true}
		assert.Equal(t, expected, got)
	})

	t.Run("withoutExamples", func(t *testing.T) {
		linStrNoExamples := `name: "single"
schemas: [{
	version: [0, 0]
	schema: {
		astring?: string
		anint:   int64 | *42
		abool:   bool
	}
}]
`

		lin := testLin(linStrNoExamples)

		sch := lin.First()
		examples := sch.Examples()
		require.NotNil(t, examples)
	})

	t.Run("withEmptyExamples", func(t *testing.T) {
		linStrNoExamples := `name: "single"
schemas: [{
	version: [0, 0]
	schema: {
		astring?: string
		anint:   int64 | *42
		abool:   bool
	}
	examples: {}
}]
`
		lin := testLin(linStrNoExamples)

		sch := lin.First()
		examples := sch.Examples()
		require.NotNil(t, examples)
	})
}

// scratch test, preserved only as a simpler sandbox for future playing with pointers, generics, reflect
func testPointerNewVar(t *testing.T) {
	type Foo struct {
		V int
	}

	mk1 := mkNew(Foo{V: 42})
	mk1v := mk1()
	if mk1v.V == 42 {
		t.Fatal("zero value should be zero")
	}

	mk1v.V = 43
	if mk1().V == 43 {
		t.Fatal("setting value of return should not influence future values")
	}

	base2 := &Foo{V: 42}
	mk2 := mkNew(base2)
	mk2v := mk2()
	if mk2v.V == 42 {
		t.Fatal("zero value should be zero")
	}

	mk2v.V = 43
	if mk2().V == 43 {
		t.Fatal("setting value of return should not influence future values")
	}
}

func mkNew[T any](t T) func() T {
	fmt.Printf("%T %v || ", t, t)
	if reflect.ValueOf(t).Kind() == reflect.Pointer {
		zt := reflect.ValueOf(t).Elem().Type()
		fmt.Printf("%T %v\n", zt, zt)
		return func() T {
			return reflect.New(zt).Interface().(T)
		}
	} else {
		zt := reflect.Zero(reflect.TypeOf(t)).Interface().(T)
		fmt.Printf("%T %v\n", zt, zt)
		return func() T {
			return zt
		}

	}
}
