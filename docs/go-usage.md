# Using Thema in (Go) Programs

In prior articles, we [wrote a `Ship` lineage in CUE](authoring.md), then made it reliably available in Go via a canonical [`LineageFactory`](https://pkg.go.dev/github.com/grafana/thema#LineageFactory) function named `ShipLineage()`. With that done, we're ready to write a program that puts Thema to work doing something useful.

Now, there are lots of kinds of programs that might use Thema. Here are a few:

* Something with a RESTful HTTP API, which needs to schematize the objects it sends and receives
* Something with configuration file(s), which govern program behavior 
* Something with SQL-shaped storage, which needs some kind of DDL/schema to define its tables 
* Something with NoSQL-shaped storage, where the absence of native database schema makes the need for app-level schema even greater
* Something with protobuf endpoints, which are intrinsically schematized but safely evolving them [is hard](https://docs.buf.build/breaking/rules)
* Something that is a backend to a frontend/browser app, and both need a common language for specifying the data they exchange
* Something that acts as a [Kubernetes Operator](https://www.redhat.com/en/topics/containers/what-is-a-kubernetes-operator), where defining evolvable schema ([CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)) is table stakes

Many of these cases have mature solutions. Some are unlikely to ever be reached by Thema, and some uses for Thema aren't represented. But all of these cases share at least one property: whatever the task at hand is, simultaneously juggling schema versions multiplies the task's complexity. We'll never get rid of the need to evolve and change our schemas, so the best outcome is encapsulating that juggling to a corner of the program, thereby allowing the rest of the program to _pretend_ that only one version exists.

This tutorial will focus on a general approach to encapsulating the problem of receiving input data, validating it, translating it, and make it available for use as a Go struct. We refer to this cluster of behavior as an **Input Kernel**.

But before we run, we should learn to walk. Input kernels are patterned clusters of behavior composed from Thema's core operations. You can [jump ahead](#input-kernels) to read about the kernels , The best way to understand what you're doing when you create an input kernel is to learn those core operations, one at a time.

## Thema Core Operations

At the start of every story involving schemas, Thema included, there exist two things:

1. Some data
2. A schema

However that schema is expressed - JSON Schema, OpenAPI, native language types, etc. - and whatever format the data is in - CSV, JSON, YAML, Protobuf, Arrow Dataframes, native language objects, etc. - the first thing we want to know is, "is the data a valid instance of the schema?"

With Thema, this question has a new dimension. Thema shifts the contract from "data must be an instance of **THIS** schema," to "data must an instance of **SOME** schema in the lineage." That suggests our validation process also may contain a search component.

We're still working with the `Ship` lineage we created over the past couple tutorials. Let's use this JSON as our input:

```json
{
    "firstfield": "foovalue"
}
```
### Load

Before we can validate, we have to get the data into the form that Thema's validation functions expect: a `cue.Value`.

The challenge here - efficiency aside - is picking from a large buffet of options. CUE's Go API has [many](https://pkg.go.dev/cuelang.org/go@v0.4.0/cue#Context.Encode) [different](https://pkg.go.dev/cuelang.org/go@v0.4.0/cue#Context.EncodeType) [ways](https://pkg.go.dev/cuelang.org/go@v0.4.0/cue#Value.FillPath) [to](https://pkg.go.dev/cuelang.org/go@v0.4.0/cue#Context.CompileString) [convert](https://pkg.go.dev/cuelang.org/go@v0.4.0/cue#Context.BuildInstance) [various](https://pkg.go.dev/cuelang.org/go@v0.4.0/encoding/json#Extract) [inputs](https://pkg.go.dev/cuelang.org/go@v0.4.0/encoding/yaml#Extract) [to](https://pkg.go.dev/cuelang.org/go@v0.4.0/encoding/jsonschema#Extract) their corresponding `cue.Value`. The right method depends on how your program has received its inputs, in what format, and whether you're trying to work with concrete data or abstract types[^cueduality], ("incomplete," in CUE parlance).

In our case, we've hand-written raw JSON in a byte slice. This requires two calls - one to extract a CUE AST-equivalent to the JSON (an `ast.Expr`), and a second to build that AST value into our universe. We'll wrap these into a function called `dataAsValue()`:

```go
package example

import (
    "testing"

	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/encoding/json"
	"github.com/grafana/thema"
)

var input = []byte(`{
    "firstfield": "foo"
}`)

func dataAsValue(lib thema.Library) cue.Value {
    // The first parameter gives the CUE evaluator a name that it will use for
    // any future errors involving the extracted data. Usually this is derived
    // from a file path, but our input is coming from an arbitrary string, so we
    // must name it ourselves.
    expr, _ := json.Extract("input", input)

    // Load our data into a CUE context-universe and return a cue.Value reference to it.
    return lib.Context().BuildExpr(expr)
}
```

With data-as-`cue.Value` prepared, we're ready to start validating. But first, let's quickly overview the types we'll be relying on.

### Type Overview

Thema's Go library presents three types for its core operations:

* [`Lineage`](https://pkg.go.dev/github.com/grafana/thema#Lineage): represents a whole lineage; what we created in previous tutorials. Closed interface.
* [`Schema`](https://pkg.go.dev/github.com/grafana/thema#Schema): represents an individual schema from a lineage. Closed interface.
* [`Instance`](https://pkg.go.dev/github.com/grafana/thema#Instance): represents data that's a valid instance of some schema from some lineage. Struct with hidden members.

These directly represent three of the [core concepts](overview.md). All of these types are closed in order to ensure that a non-nil variable of the type is constructed in a manner that is ready for use, and confers Thema's [guarantees](invariants.md).

These types are connected through methods that represent their well-defined relations. You can look up a particular `Schema` from a `Lineage` by version number, or go from a `Schema` to its `Lineage`. An `Instance` can return its `Schema`, but that's a one-way trip - `Schema` do not keep an internal index of validated `Instance`s. The graph of connected objects is always limited to those associated with a single lineage.

### Hand-pick and validate

The simplest approach to validation is to pretend that Thema is like any old schema system, and manually select one schema at a time to work with. We'll express this using standard Go tests, as it makes it easiest to see the output on your machine.

Let's start with a simple test to check what we already know - our input is an instance of schema 0.0, but not of schema 1.0. For that, we'll rely on `Lineage.Schema()` to retrieve a particular schema, and `Schema.Validate()` to check the data.

```go
package example

import (
    "testing"

    "cuelang.org/go/cue/cuecontext"
    "github.com/grafana/thema"
)

var lib thema.Library = thema.NewLibrary(cuecontext.New())
var shiplin thema.Lineage

func init() {
    var err error
    if shiplin, err = ShipLineage(lib); err != nil { panic(err) }
}

func TestHandpickValidation(t *testing.T) {
    // Ask the lineage for the schema with version 0.0. An error can only happen
    // if you request a schema version that doesn't exist.
    sch, _ := shiplin.Schema(thema.SV(0, 0))
    _, err := sch00.Validate(dataAsValue(lib))
    // Our input is valid according to schema 0.0, so there should be no error
    if err != nil {
        t.Fatal(err)
    }
}
```

Here, we've hand-picked the schema version we want to validate against - `0.0`, which every lineage is guaranteed to contain. The [`LatestVersion()`](https://pkg.go.dev/github.com/grafana/thema#LatestVersion) and [`LatestVersionInSequence()`](https://pkg.go.dev/github.com/grafana/thema#LatestVersionInSequence) functions provide fuzzier version selection logic. But allowing only one schema version as input somewhat defeats the purpose of using Thema in the first place. Ideally, we'd have something more dynamic.

### Search by validity

The first step to simplifying the ingestion of data from multiple possible schemas is to stop treating them individually. (It's like ["cattle, not pets"](http://cloudscaling.com/blog/cloud-computing/the-history-of-pets-vs-cattle/), but for schema!) [`Lineage.ValidateAny()`](https://pkg.go.dev/github.com/grafana/thema#Lineage) gives us what we want, here: instead of having to preselect the schema we want to validate against, we only want to know if the data is valid against _any_ schema in our lineage.

```go
package example

import (
    "testing"

    "cuelang.org/go/cue/cuecontext"
    "github.com/grafana/thema"
)

var lib thema.Library = thema.NewLibrary(cuecontext.New())
var shiplin thema.Lineage

func init() {
    var err error
    if shiplin, err = ShipLineage(lib); err != nil { panic(err) }
}

func TestSearchByValid(t *testing.T) {
    inst := shiplin.ValidateAny(dataAsValue(lib))
    // The returned schema is the one with the smallest version number for which
    // the data is valid. Failure to find any schema for which the data is
    // valid results in a nil return. 
    if inst == nil {
        t.Fatal("expected input data to validate against schema 0.0")
    }
}
```

Of course, this approach presents a new question. Any of the schemas could have matched, but we don't actually know which one did. To find out, we ask the instance for its schema, then ask that schema for its version.

```go
func TestSearchByValid(t *testing.T) {
    inst := shiplin.ValidateAny(dataAsValue(lib))
    if inst == nil {
        t.Fatal("expected input data to validate against schema 0.0")
    }
    // Figure out which schema version validated by getting the schema of the
    // instance, then asking the schema for its version.
    fmt.Println(inst.Schema().Version()) // 0.0
}
```

OK, now we know what version we validated against. But relying on search means we just fanned out to accept every possible schema as input. That's the opposite of the outcome we want - writing our programs against a single version of schema - so we need to fan back in.

### Translate

Fanning in to a single version of our schema means putting Thema's system of lenses and translation to work. Given an instance of `Ship`, regardless of what version it starts at, we want to translate to one known, fixed version throughout our program. (This is analogous to pinning a dependency's version with a traditional package manager.) For now, let's put it in a package variable called `targetVersion`, and pick `1.0`.

Calling `Translate()` on an instance will produce two values: a new instance valid with respect to the schema version that was specified, and any lacunas that the translation process produced. And our `Ship` lineage [does emit one](https://github.com/grafana/thema/blob/main/docs/authoring.md#emitting-a-lacuna), because we had to put that placeholder `-1` value in for `secondfield`.

```go
var targetVersion = thema.SV(1, 0)

func TestSearchByValid(t *testing.T) {
    inst00 := shiplin.ValidateAny(dataAsValue(lib))
    if inst == nil {
        t.Fatal("expected input data to validate against schema 0.0")
    }

    inst10, lacunas := inst.Translate(targetVersion)
	byt, _ := json.MarshalIndent(map[string]interface{}{
		"inst0.0": inst00.UnwrapCUE(),
		"inst1.0": inst10.UnwrapCUE(),
		"lacunas": lacunas.AsList(),
	}, "", "    ")
	fmt.Println(string(byt))
}
```

Output:

```json
{
    "inst0.0": {
        "firstfield": "foo",
    },
    "inst1.0": {
        "firstfield": "foo",
        "secondfield": -1
    },
    "lacunas": [
        {
            "targetFields": [
                {
                    "path": "secondfield",
                    "value": null
                }
            ],
            "type": 0,
            "message": "-1 used as a placeholder value - replace with a real value before persisting!"
        }
    ]
}
```

We've got our `Ship` instance that's valid with respect schema `1.0`!

We also have a lacuna, telling us that the contents of `secondfield` is a placeholder value. In a real program, we'd want to do something about this. But working with lacuna is its own, complex topic, so we're going to ignore it for now.

### Decode

If we're planning on actually working with this `Ship` instance in our Go program, there's one last step to take: populate a Go type with our data.

First, we need a Go type. For now[^gocodegen], we'll need to hand-write a `Ship` struct:

```go
// CUE piggybacks `json` struct tags.
type Ship struct {
	Firstfield  string `json:"firstfield`
	Secondfield int    `json:"secondfield`
}
```

Finally, we'll expand our test to load the contents of our `*Instance` into a Go variable of this new `Ship` type. (This is somewhat analogous to JSON unmarshalling, but in CUE is called `Decode`.)

```go
func TestSearchByValid(t *testing.T) {
    inst00 := shiplin.ValidateAny(dataAsValue(lib))
    if inst == nil {
        t.Fatal("expected input data to validate against schema 0.0")
    }

    inst10, _ := inst.Translate(targetVersion)

    var ship Ship
    inst10.UnwrapCUE().Decode(&ship)
    fmt.Printf("%+v\n", ship) // "{Firstfield:foo Secondfield:-1}"
}
```

With our native Go type populated, our program is now ready to forget that Thema exists and act like any other Go program.

* Verify that the Go `Ship` struct type is compatible with what's declared in schema `1.0`.

## Input Kernel

Manually stitching together a Thema-based input processing flow can be done. Clearly - we've just done it. But all we've really made is function calls scattered across tests, rather than a nice, tight system. Ideally, there'd be an approach that miimally distracts us from the harder problem: composing Thema into larger systems. Start with a `[]byte` of input data, end with our desired Go type, all in a minimal structure made from the answer to a few high-level questions:

* Which lineage are we using?
* What data format are we expecting as input?
* Which schema version are we targeting?
* What Go type do we want our data to end up in?

Enter, [`InputKernel`](https://pkg.go.dev/github.com/grafana/thema/kernel#InputKernel).

Thema's kernels encapsulate common patterns for getting data into and out of a running program. The `InputKernel` does this for the pattern we just wrote out manually. To create one, we answer those four questions:

```go
func TestKernel(t *testing.T) {
    lib := thema.NewLibrary(cuecontext.New())
    lin, _ := ShipLineage(lib)

    k, _ := kernel.NewInputKernel(kernel.InputKernelConfig{
        // "Which lineage are we using?"
		Lineage:     lin,
        // "What data format are we expecting as input?"
		Loader:      kernel.NewJSONDecoder("shipinput.json"),
        // "What schema version are we targeting?"
		To:          thema.SV(1, 0),
        // "What Go type do we want our data to end up in?"
		TypeFactory: func() interface{} { return &Ship{} },
	})

    val, _, _ := k.Converge([]byte(`{
        "firstfield": "foo"
    }`))
    var ship *Ship = val.(*Ship)
    fmt.Printf("%+v\n", ship) // "{Firstfield:foo Secondfield:-1}"
}
```

`InputKernel.Converge()` returns `interface{}` (which wraps a `*Ship` returned from `TypeFactory`), meaning that we've lost conventional Go type safety[^generics]. But there's a trick: `NewInputKernel()` validates that the return from `TypeFactory` is valid with respect to the schema specified by `To`, and errors out otherwise.

Enforcing type-checking in `NewInputKernel()` means that every instance of `InputKernel` has certain runtime type safety guarantees:

* The concrete type returned from `TypeFactory` can correctly represent any data valid with respect to the `To` CUE schema
* A non-error return from `Converge()` will always return the same concrete type returned from `TypeFactory`

These guarantees, alongside the fact that `InputKernel.Converge()` is stateless, makes all this a good candidate for wrapping everything up into a function `JSONToShip()`, that makes sense to export alongside our `Ship` Go type:

```go
package example

// Ship is a Thesian data vessel. Version 1.0
type Ship struct {
	Firstfield  string `json:"firstfield`
	Secondfield int    `json:"secondfield`
}

// The single, canonical Thema schema version of Ship the program is currently
// written against. Real programs should figure out how and where to keep a
// runtime-immutable catalog of these.
var shipVersion = thema.SV(1, 0)

// JSONToShip converts a byte slice of JSON data containing a single instance of
// ship valid against any schema
func JSONToShip(data []byte) (*Ship, thema.TranslationLacunas, error) {
	ship, lac, err := jshipk.Converge(data)
	if err != nil {
		return nil, nil, err
	}
	return ship.(*Ship), lac, nil
}

// ShipVersion reports which Ship Thema schema this program is currently written
// against.
func ShipVersion() thema.SyntacticVersion {
	return shipVersion
}

// Because there is no conceivable case in which ShipVersion should change at
// runtime (the definition of the Ship type would have to change, thereby
// requiring recompilation), keeping a kernel in package state and computing it
// in init() is perfectly fine.
var jshipk kernel.InputKernel

func init() {
	// Creating a one-off cue.Context for this purpose is acceptable because
	// there's no possibility of having to combine with externally-defined
	// cue.Values that may have come from a different cue.Context. However, real
	// programs may find some performance benefit due to reuse if they create a
	// package with a singleton Library.
	lib := thema.NewLibrary(cuecontext.New())

	lin, err := ShipLineage(lib)
	if err != nil {
		panic(err)
	}

	jshipk, err = kernel.NewInputKernel(kernel.InputKernelConfig{
		Lineage:     lin,
		Loader:      kernel.NewJSONDecoder("shipinput.json"),
		To:          shipVersion,
		TypeFactory: func() interface{} { return &Ship{} },
	})
	if err != nil {
		// Panic here guarantees that our program cannot start if the Go ship type is
		// not aligned with the shipVersion schema from our lineage
		panic(err)
	}

}
```

`JSONToShip()` composes the entire Thema stack together into a single input processing function. To get there, we've made some opinionated choices that won't be appropriate for all use cases. But it's built from small, exported, composable parts, so adapting it to other use cases ought not be prohibitively difficult.

### Use case: Application config files

TODO this one is trivial

### Use case: HTTP frameworks

TODO trickier than config because reusable middleware requires generics, and even then it's not simple `http.Handler`, so applying `JSONToShip()` probably has to happen at the tail end of `http.Handler`

TODO mention how HTTP Request headers can be used to decide which version to translate the `Ship` back to on egress

## Wrap-up

This tutorial illustrated how to take the [`LineageFactory`](https://pkg.go.dev/github.com/grafana/thema#LineageFactory) called `ShipLineage` that we created in the [last tutorial](go-mapping.md), and use it to create an `InputKernel` around a Go `Ship` type that can handle valid input of any ship schema in our lineage and land on a populated instance of our `Ship` type.

The [next (TODO)](lacuna-programming.md) tutorial will show how to add the final, tidy layer for working with Thema in programs: handling lacunas that are emitted from `Translate()`.

[^cueduality]:
    We've seen `cue.Value` before, when creating the lineage factory in the previous tutorial. That one represented our whole lineage, but this one represents JSON data. It may seem odd that both abstract schema and concrete JSON are represented in the same way. And indeed, if you end up going deeper with CUE's Go API, keeping track of exactly what's represented by the `cue.Value` you're working with gets challenging.
    
    But the lack of distinction between schema and data here isn't just a quirk. It's a direct outgrowth of CUE's most foundational property: [types are values](https://cuelang.org/docs/concepts/logic/#types-are-values).

[^gocodegen]:
    CUE doesn't yet have standard library support for generating Go structs from CUE values. When it does, it will slot in nicely here. Even without codegen, we don't compromise on correctness - we can validate hand-written types against the CUE schema, as well.

[^generics]:
    Yes, this should be a case that generics can address - though they still won't be able to cover everything CUE schemas provide (defaults, value bounds, precise optionality semantics).
