# Using Thema in Programs

In prior articles, we [wrote a `Ship` lineage in CUE](authoring.md), then made it reliably available in Go via a canonical [`LineageFactory`](https://pkg.go.dev/github.com/grafana/thema#LineageFactory) function named `ShipLineage()`. With that done, we're ready to write a program that puts Thema to work doing something useful.

Now, there are lots of kinds of programs that might use Thema. Here are a few:

* Something with a RESTful HTTP API, which needs to schematize the objects it sends and receives
* Something with configuration file(s), which govern program behavior 
* Something with SQL-shaped storage, which needs some kind of DDL/schema to define its tables 
* Something with NoSQL-shaped storage, where the absence of native database schema makes the need for app-level schema even greater
* Something with protobuf endpoints, which are intrinsically schematized but safely evolving them [is hard](https://docs.buf.build/breaking/rules)
* Something that is a backend to a frontend/browser app, and both need a common language for specifying the data they exchange
* Something that acts as a [Kubernetes Operator](https://www.redhat.com/en/topics/containers/what-is-a-kubernetes-operator), where defining evolvable schema ([CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)) is table stakes

Many of these cases have mature solutions. Some are unlikely to ever be reached by Thema, and some uses for Thema aren't represented. But all of these cases share at least one property: whatever the task at hand is, simultaneously juggling schema versions multiplies the task's complexity. Since we'll never get rid of the need to evolve and change our schemas, the best outcome is encapsulating that juggling to a corner of the program, thereby allowing the rest of the program to _pretend_ that only one version exists.

This tutorial will focus on a general approach to encapsulating the problem of receiving input data, validating it, translating it, and make it available for use as a Go struct. We refer to this cluster of behavior as an **Input Kernel**.

But before we run, we must learn to walk. Input kernels are patterned clusters of behavior composed from Thema's core operations. The best way to understand what you're doing when you create an input kernel is to learn those core operations, one at a time.

## Thema Core Operations

At the start of every story involving schemas, Thema included, there exist two things:

1. Some data
2. A schema

However that schema is expressed - JSON Schema, OpenAPI, native language types, etc. - and whatever format the data is in - CSV, JSON, YAML, Protobuf, Arrow Dataframes, native language objects, etc. - the first thing we want to know is, "is the data a valid instance of the schema?"

With Thema, this question has a new dimension. Thema shifts the contract from "data must be an instance of **THIS** schema," to "data must an instance of **A** schema in the lineage." That suggests our validation process also may contain a search component. 

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

With data-as-`cue.Value` prepared, we're ready to start validating.

### Hand-pick and validate

The simplest approach to validation is to pretend that Thema is like any old schema system, and manually select one schema at a time to work with. We'll express this using standard Go tests, as it makes it easiest to see the output on your machine.

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

func TestManual(t *testing.T) {
    sch00 := shiplin.Schema(thema.SV(0, 0))
}
```
### Search by validity

### Translate

## Input Kernel

[^cueduality]:
    We've seen `cue.Value` before, when creating the lineage factory in the previous tutorial. That one represented our whole lineage, but this one represents JSON data. It may seem odd that both abstract schema and concrete JSON are represented in the same way. And indeed, if you end up going deeper with CUE's Go API, keeping track of exactly what's represented by the `cue.Value` you're working with gets challenging.
    
    But the lack of distinction between schema and data here isn't just a quirk. It's a direct outgrowth of CUE's most foundational property: [types are values](https://cuelang.org/docs/concepts/logic/#types-are-values).