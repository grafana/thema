# Encapsulating Lineages in Go

Once we know how to [write a Thema lineage in CUE](authoring.md), a common next step is to make the lineage available for use in a general-purpose programming language, like Go. No matter what we'd like this Go program to do with our lineages, we must first get the CUE text of our lineage declaration - often referred to as "the bytes" in this doc - into Go, and load it into the [types exported by the `thema` package](https://pkg.go.dev/github.com/grafana/thema).

Thema's Go types are intentionally designed to limit extension: exported interfaces and structs with some or all unexported members. Thema's value as a schema system derives primarily from its [guarantees](invariants.md), and those guarantees must be available in Go, with the `Lineage` type as the entry point. To its consumers, `Lineage` should be a powerful, reliable abstraction: if some Go code has a non-`nil` variable of type `thema.Lineage`, all of Thema's (implemented) guarantees apply, unconditionally.

That's a serious guarantee, especially given that responsibility for fulfilling it will fall to the Thema author. Hopeium won't cut it. For Thema guarantees to actually hold in the wild - where `Lineage` instances burst forth from Go code written by those of us mortals who haven't _quite_ gotten around to finishing our maths PhD _just_ yet - it must be near-impossible to produce a `Lineage` that doesn't keep Thema's promises. As we'll see, Thema approaches this by making [`BindLineage()`](https://pkg.go.dev/github.com/grafana/thema#BindLineage) a verification choke point: it's the only way to create a `Lineage`, and will error out if provided raw CUE that does not constitute a valid lineage.

With this in mind, this tutorial puts us in the role of the lineage author with the goal of creating a standalone Go package that can return an instance of `thema.Lineage` from the `Ship` lineage we [previously](authoring.md) created in CUE.

## Environment setup

Let's set up a local environment so you can create these components yourself. You'll need [Go](https://go.dev/dl/) and [CUE](https://cuelang.org/docs/install/) installed.

Initialize a Git repository in a new directory:

```bash
git init thema-example
```

Next, initialize a Go module and a CUE module, both using the same path.

```bash
MODPATH="github.com/example/thema"
go mod init $MODPATH
cue mod init $MODPATH
```

Finally, grab [`ship.cue`](https://github.com/grafana/thema/blob/main/docs/ship.cue):

```bash
curl https://raw.githubusercontent.com/grafana/thema/main/docs/ship.cue > ship.cue
```

We'll use the `Ship` lineage throughout this tutorial.

## Goal: The Lineage Factory

For each lineage you create in CUE, the recommended, idiomatic approach is to export a single Go function that satisfies the [`LineageFactory`](https://pkg.go.dev/github.com/grafana/thema#LineageFactory) type. The lineage factory function will be the canonical way of accessing your lineage in any Go program. It should follow a naming pattern:

```go
func Lineage<Name> (lib thema.Library) (thema.Lineage, error) { ... }
var _ thema.LineageFactory = Lineage<Name>
```

For Go packages that clearly correspond to a single lineage declaration, the lineage name may be omitted:

```go
func Lineage (lib thema.Library) (thema.Lineage, error) { ... }
var _ thema.LineageFactory = Lineage
```

This function should encapsulate the logic for getting the CUE bytes, building the `Lineage` object, etc. The remainder of this document deals with filling in the `...`.

### Idiomatic Thema

Exporting exactly one lineage factory per declared lineage is almost always preferable. There are some other idiomatic approaches to providing Thema that are less universal, but should be followed when possible:

* Colocate the Go package containing the lineage factory in the same directory as the `.cue` file containing the lineage declaration.
* Use Go 1.16 [embedding](https://pkg.go.dev/embed) to bind `.cue` files to the package containing your lineage factory.

Thema's exemplars package, which we'll refer to continuously throughout this doc, illustrates most of these idioms.

## Map the bytes 

The first step is getting all the `.cue` files we need into our Go program. The obvious file is `ship.cue` - but we _also_ need the upstream thema `.cue` files we vendored, which are necessary for the CUE runtime to satisfy the `import "github.com/grafana/thema"` statement in `ship.cue`.

In a new `lineage.go` file, we'll embed our `.cue` files into an `embed.FS`. Then we'll use the [`InstancesWithThema`](https://pkg.go.dev/github.com/grafana/thema/load#InstancesWithThema) helper function[^loaderhelper] to load the raw CUE files we embedded. 

```go
package example

import (
    "embed"

    "github.com/grafana/thema/load"
)

//go:embed ship.cue cue.mod
var modFS embed.FS

func loadLineage() (cue.Value, error) {
    // "." loads the root directory of the modFS, where our our ship.cue is
    // located. If no args are passed to InstancesWithThema's third variadic
    // parameter, "." is used as the default.
    insts, err := load.InstancesWithThema(modFS, ".")
    if err != nil {
        return cue.Value{}, err
    }
}
```

_NOTE: this code won't compile. It will by the end._

Making CUE files available from Go is typically a two-step process: first, you load the raw files from disk - as we've done above - which performs basic processing and validation, and results in [`[]*build.Instance`](https://pkg.go.dev/cuelang.org/go@v0.4.0/cue/build#Instance). These instances must then be loaded into a [`cue.Context`](https://pkg.go.dev/cuelang.org/go@v0.4.0/cue#Context) - the top-level container for the graph of values maintained by CUE's runtime. (`cue.Context` is very different from stdlib Go [context](https://pkg.go.dev/context) - it has nothing to do with timeouts or cancellation.)

This matters for how we create our lineage factory because higher-level Thema operations over lineages and schemas are built from lower-level CUE operations over [cue.Values](https://pkg.go.dev/cuelang.org/go@v0.4.0/cue#Value). All the Go types Thema provides are largely just handles pointing to some particular `cue.Value`. To work properly, those `cue.Value`s all have be in the same universe/runtime/`cue.Context`. Let's create one, then build our `[]*build.Instance` into the `cue.Context` universe, resulting in a useful thing: a `cue.Value`.

```go
package example

import (
    "embed"

	"cuelang.org/go/cue/cuecontext"
    "github.com/grafana/thema/load"
)

//go:embed ship.cue cue.mod
var modFS embed.FS

func loadLineage() (cue.Value, error) {
    insts, err := load.InstancesWithThema(modFS)
    if err != nil {
        return cue.Value{}, err
    }

    ctx := cuecontext.New()
    val := ctx.BuildInstance(insts[0]) // above args to InstancesWithThema() guarantee len(insts) == 1
}
```

Managing `cue.Context` is tricky, though. Creating a one-off context-universe for a value that's eventually supposed to be reusable elsewhere is almost certainly wrong. It would severely limit the utility of the `Lineage` we'll eventually return for doing operations against data.

Instead, we should take a `cue.Context` as an argument. But that's also complicated, because Thema needs not only a shared universe, but a shared universe _that has loaded Thema's pure CUE logic_. 

To simplify this, Thema has a [`Library`](https://pkg.go.dev/github.com/grafana/thema#Library) type. `Library` is a thin wrapper around `cue.Context` - just some helper methods, including one to retrieve the context, with the assurance that Thema is loaded and addressable within that context. We'll see how this gets used a little later, but for now, let's update our function:

```go
package example

import (
    "embed"

	"cuelang.org/go/cue/cuecontext"
    "github.com/grafana/thema"
    "github.com/grafana/thema/load"
)

//go:embed ship.cue cue.mod
var modFS embed.FS

func loadLineage(lib *thema.Library) (cue.Value, error) {
    insts, err := load.InstancesWithThema(modFS)
    if err != nil {
        return cue.Value{}, err
    }

    val := lib.Context().BuildInstance(insts[0])
}
```

Much better. Now, with a `cue.Value` in hand, the first critical stage of mapping is complete: we've mapped raw file bytes into a working object in a CUE context-universe provided by the caller.

## Bind a `Lineage`

We have a `cue.Value` - great! But there's a low ceiling on how much we can actually say about this freshly-loaded `val`. In our particular case, we can be sure that `val` points to just the contents of `ship.cue`, because that's literally the only file in our CUE module. But in general, each element returned from `InstancesWithThema` can represent a pile of files from a hierarchy of directories, all automatically unified together, according to [CUE's rules for filesystem organization](https://cuelang.org/docs/references/spec/#modules-instances-and-packages).

All we can say for sure is that `val` represents a bunch of syntactically valid CUE statements. That's a far cry from the [strong guarantees](invariants.md) lineages are supposed to provide. That's what we do next, in two phases:

1. Retrieve the particular `cue.Value` that's actually supposed to be the `#Lineage` of our `Ship`
2. Verify that it's a valid lineage, and wrap it in our Go `Lineage` type

### Retrieval

In CUE, as in most data-centric langauges, data is structure is code is design. As such, our simple `ship.cue` file already made a design choice by declaring its lineage under the name `lin`:

```cue
package example

import "github.com/grafana/thema"

lin: thema.#Lineage
lin: Name: "Ship"
lin: Seqs: [
// ...
```

Our Go program now must mirror that design choice, traversing to the value we know actually represents our lineage:

```go
package example

import (
    "embed"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
    "github.com/grafana/thema"
    "github.com/grafana/thema/load"
)

//go:embed ship.cue cue.mod
var modFS embed.FS

func loadLineage(lib thema.Library) (cue.Value, error) {
    insts, err := load.InstancesWithThema(modFS)
    if err != nil {
        return cue.Value{}, err
    }

    val := lib.Context().BuildInstance(insts[0])
    return val.LookupPath(cue.MakePath(cue.Str("lin"))), nil
}
```

Presto! We've retrieved the `cue.Value` that represents our lineage. But let's have a moment of consideration before continuing.

That hardcoded `"lin"` string tightly and implicitly couples our Go logic with this arbitrary choice made in CUE. If we change the `lin` path to something else, how would the above code fail?

The simplicity of our `ship.cue` and example module as a whole suggests that it might've been better to omit the `lin` and just make the whole `ship.cue` file our lineage:

```cue
package example

import "github.com/grafana/thema"

thema.#Lineage
Name: "Ship"
Seqs: [
// ...
```

If our CUE had this structure, then the `val` returned from `ctx.BuildInstance` _would be_ our `Ship` lineage - no need to `LookupPath()` before returning. Nice!

It's tempting to imagine "instance == lineage" as another idiomatic best practice. And it would be great to have a general rule - anything that further structures the relationship between CUE and Go code makes it easier to reason about. Unfortunately, this rule doesn't generalize well. 

Thema's [exemplars](https://github.com/grafana/thema/tree/main/exemplars), for example, are [part of a larger type](https://github.com/grafana/thema/blob/main/exemplars/constraint.cue#L6) that acts partially as testing harness and partially as basis for documentation. The whole package contains multiple lineages, and [is constrained](https://github.com/grafana/thema/blob/main/exemplars/constraint.cue#L17) to expect each named exemplar harness as a top-level field, like our `lin`.

There's also entirely other ways of marking which CUE values are supposed to be lineages. [Attributes](https://github.com/grafana/thema/blob/main/exemplars/constraint.cue#L17), for example, could be a clean way to indicate to generic tooling (triggered e.g. via a `//go:generate` directive) that a particular lineage should be translated to another language[^cuetsy]:

```cue
package example

import "github.com/grafana/thema"

lin: thema.#Lineage @thematranslate(protobuf) // purely illustrative; not real/supported by any tool
lin: Name: "Ship"
lin: Seqs: [
// ...
```

There's nothing wrong or inadvisable about either of these approaches. If anything, they demonstrate the importance of keeping the Go loading layer flexible in order to avoid unnecessary, composition-limiting constraints on what's done in CUE.

Let's tidy up our Go program into terse, final form, without getting rid of the `lin` from `ship.cue`:

```go
package example

import (
    "embed"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
    "github.com/grafana/thema"
    "github.com/grafana/thema/load"
)

//go:embed ship.cue cue.mod
var modFS embed.FS

func loadLineage(lib thema.Library) (cue.Value, error) {
    if insts, err := load.InstancesWithThema(modFS); err == nil {
        return lib.Context().BuildInstance(insts[0]).LookupPath(cue.ParsePath("lin")), nil
    } else {
        return cue.Value{}, err
    }
}
```

Now, there's still the nagging question about where the failure happens if the `lin` path changes. But that's a verification issue - which is what's up next.

### Build and Verify

We're ready to write our lineage factory func, `LineageShip()`!

```go
package example

import (
    "embed"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
    "github.com/grafana/thema"
    "github.com/grafana/thema/load"
)

//go:embed ship.cue cue.mod
var modFS embed.FS

func loadLineage(lib thema.Library) (cue.Value, error) {
    if insts, err := load.InstancesWithThema(modFS); err == nil {
        return lib.Context().BuildInstance(insts[0]).LookupPath(cue.ParsePath("lin")), nil
    } else {
        return cue.Value{}, err
    }
}

// LineageShip constructs a Go handle representing the Ship lineage.
func LineageShip(lib thema.Library) (thema.Lineage, error) {
    linval, err := loadLineage(lib)
    if err != nil {
        return nil, err
    }
    return thema.BindLineage(linval, lib)
}
var _ thema.LineageFactory = LineageShip // Ensure our factory fulfills the type
```

Well, that was anticlimatic. `thema.BindLineage()` did all the work!

But that's the point: as the author of the `Ship` lineage, we want to offer it up as an instance of the Go `Lineage` type. Consumers of `LineageShip()` want certainty that the return value faithfully upholds the guarantees that Thema promises about lineages in general. If Thema authors were forced to make a lot of choices in their lineage factorys, it would introduce room for error in the delivery of those guarantees. Instead, responsibility for verification is delegated[^cuevalidity] to `BindLineage()`.

So, to check whether our `Ship` lineage is valid, all we have to do check the `error` return of our lineage factory. A trivial test is sufficient[^panic]:

```go
package example

import (
    "testing"

	"cuelang.org/go/cue/cuecontext"
    "github.com/grafana/thema"
)

func TestShipIsValid(t *testing.T) {
    if _, err := LineageShip(thema.NewLibrary(cuecontext.New())); err != nil {
        t.Fatal(err)
    }
}
```

Our `Ship` lineage is now wrapped up in a reliable package[^pubretrieve], ready to be consumed.

### Advanced: Additional Verification

`BindLineage()` provides basic lineage validity guarantees. However, we may have more things we want to verify about `Ship` - and if so, `LineageShip()` is the place to do it, _after_ a non-error return from `BindLineage()`.

TODO

## Wrap-up

This tutorial illustrated how, as a Thema lineage author, we take a CUE `#Lineage` and make it available to Go programs as a [`Lineage`](https://pkg.go.dev/github.com/grafana/thema#Lineage) via a standard [`LineageFactory`](https://pkg.go.dev/github.com/grafana/thema#LineageFactory) function.

In the [next tutorial](go-usage.md), we'll trade our Thema author hat for a Thema consumer hat, and show how to write a Go program that uses the Thema `Lineage` returned from `LineageShip()` to be written against just one version of `Ship`, but be able to handle `Ship`s in any form specified in the lineage.

[^loaderhelper]:
    `InstancesWithThema` abstracts over [`load.Instances`](https://pkg.go.dev/cuelang.org/go@v0.4.0/cue/load#Instances), which offers far more [options](https://pkg.go.dev/cuelang.org/go@v0.4.0/cue/load#Config) than are usually needed for Thema. It's expected that some more complex cases will not fit into `InstancesWithThema`; in such case, plan to write your own loader-helper.

[^cuetsy]:
    Grafana currently relies on an attribute-driven mechanism for translating Thema schemas to TypeScript using the [cuetsy](https://github.com/grafana/cuetsy) library. Current usage is different than the protobuf example here because cuetsy is not Thema-aware, and authors therefore apply attributes to individual schema within lineages.

[^cuevalidity]:
    The goal is that all constraints necessary for invariant enforcement on lineages are expressed natively in CUE. However, some of that enforcement is currently performed by `BindLineage()` in Go, because it's not (yet) possible to express the necessary constraints natively in CUE.
    
    Today, lineage authors delegate verification to `BindLineage()`, resulting in uniform invariant enforcement across all Go `Lineage` instances. But when all necessary constraints are expressed in CUE, verification can shift left (up?). `BindLineage()` will delegate enforcement to CUE itself, becoming a passthrough more akin to `LineageShip()`, and guarantee uniformity will naturally extend not just to a Go `Lineage`, but to any analogous construct in the Thema bindings for another language (that has a CUE evaluator).

[^panic]:
    Arguably, you could call `LineageShip()` in an `init()` function, and `panic()` on error. `panic()` is usually reserved in Go for unrecoverable errors, and a lineage failing to load is unrecoverable: the only possible sources of failure are a) a buggy CUE evaluator, b) backwards incompatible changes in CUE itself, or c) the input CUE is not a valid `#Lineage`. As long as the bytes representing the input CUE arrive through reliable transport (e.g. from a colocated file `//go:embed`-ed in the binary), no remediation is possible within the scope of the running program.

[^pubretrieve]:
    Barring bugs, the only systemic failure mode is a side effect of what made this tutorial relatively uncomplicated: embedding. Embedding CUE files into Go makes them reliably available, but it also couples the set of known schema in the lineage to the version of the Go package into which they're embedded. If we add a new schema to `Ship`, Go programs in other repos/modules won't see it until their `go.mod` is updated and they're recompiled. The central Thema guarantee - translatability of any valid instance to the schema version the program expects - would be lost until recompilation.

    Ideally, package version and lineage state would be decoupled, and lineage state updates handled at runtime. Mapping from CUE `#Lineage` to (lang) `Lineage`-equivalent is something that programs could do continuously in the background by periodically polling a lineage registry service over HTTP, and integrating updates into the `Lineage`-equivalent. Recompilation would no longer be needed; evolutionary independence would be restored.

    In theory, [append-only immutability constraints](invariants.md) on published lineages should make this straightforward, to the point where any language's Thema bindings could encapsulate the polling and hot-updating of the in-language `Lineage` handle into a single function, similar to what `BindLineage()` does today. (In fact, the two are likely to be complementary: start by building from reliably-available embedded, local CUE bytes, then kick off a background thread that continuously polls a server for newer bytes.) A [registry and publishing flow](https://github.com/grafana/thema/issues/6) are prerequisite to a fully general solution, but in the meantime, org-by-org one-offs should be feasible in a pinch.