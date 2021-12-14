# Basic Scuemata Writing

The first, most basic task to be done with scuemata is to write a lineage.

Writing lineages is fundamentally similar to any other schema-writing system: you're defining a specification for what data is supposed to look like. Two things are different with scuemata:

1. You're writing schemas in CUE, rather than whatever other language you may be accustomed to
2. You're defining those schema _within_ a larger, well-defined structure - the lineage - which groups those schema together and enforces certain requirements.

A primer on writing CUE is out of scope; for that, the [official CUE tutorials](https://cuelang.org/docs/tutorials/) and [CUE Playground](https://tip.cuelang.org/play/?id=#cue@export@cue) are a good place to start. This tutorial focuses on the second item: expressing collections of schemas as a valid lineage.

## Scaffolding

Scuemata lineages require only two fields to be explicitly defined: `Name`, and `Seqs`. 

`Name` is the identifier for the thing schematized by the lineage. This should be a simple name, not a fully-qualified one - that's for later. We'll call our thing `"OurObj"`.

`Seqs` contains the list of all sequences of all schemas within the lineage, and the lenses that map between them. It's basically a two-dimensional array.

```cue
import "github.com/grafana/scuemata"

lin: scuemata.#Lineage 
lin: Name: "OurObj"
lin: Seqs: [
    {
        schemas: []
    }
]
```

To be a valid lineage, there must be at least one sequence in `Seqs`, which in turn must contain at least one schema in its `schemas` list. That means this isn't actually a valid lineage. Rather, this is just the minimum necessary structure to begin defining a lineage. (Note that the outermost `lin` can be omitted, in which case the entire file is the lineage.)

It's essential to the reliability of scuemata that supporting tooling refuses to work with invalid lineages, similar to a failed type check. Consequently, attempting to `cue eval` or [build the lineage in Go](https://pkg.go.dev/github.com/grafana/scuemata#BuildLineage), or any program built atop similar functionality, will ([should](TODO)) fail with a complaint about the length of `schemas` being less than 1.

## Defining a Schema

Each element in the `schemas` list is treated as a single schema. By default, these schema definitions can be any CUE value, from a simple primitive `bool` to a highly complex, deeply nested object. Complex CUE constraints may also be used, though [keep in mind this may limit translatability](TODO).

Lineages should be sealed structures, as scuemata invariants only hold in the context of immutable schema declarations. Referencing CUE values from externally imported CUE modules within scuemata schema can undermine this, as those values may change when the version of the imported module is changed.

Let's define our first schema as an object containing a single field named `firstfield`, which must be of type `string`.

```cue
import "github.com/grafana/scuemata"

lin: scuemata.#Lineage 
lin: Name: "OurObj"
lin: Seqs: [
    {
        schemas: [
// Scaffold ∧ | ∨ Schema
            {
                firstfield: string
            }
// Scaffold ∨ | ∧ Schema
        ]
    }
]
```

And that's it - we now have a valid scuemata lineage, containing a single schema.

## Adding More Schemas

The schema we wrote isn't terribly exciting. We'd like to add to it.

When writing real scuemata, the question of whether to define a new schema or make additions directly to the existing one should be wholly determined by whether the latest existing schema has been published, as published schema must be immutable. That makes the definition of "published" quite important.

For this tutorial, we'll sidestep the issue by assuming that publication has happened. Therefore, making changes entails creating a new schema. Let's add one more field, `secondfield`, which must be an `int`.

_Note: in scuemata, schema version is determined by its position within the two-dimensional array structure of `Seqs`, rather than through arbitrary choice by the author. These structurally-determined version numbers are indicated as comments._

```cue
import "github.com/grafana/scuemata"

lin: scuemata.#Lineage 
lin: Name: "OurObj"
lin: Seqs: [
    {
        schemas: [
            { // 0.0
                firstfield: string
            },
            { // 0.1
                firstfield: string
                secondfield: int
            }
        ]
    }
]
```

This isn't a valid lineage, though. `secondfield` is required and lacks a default, which means that valid instances of `0.0` will be invalid with respect to `0.1`. Our change is backwards incompatible, which breaks the rules of scuemata.

There are three possible ways to fix this: make `secondfield` optional, give it a default, or start a new sequence.

## Optional Fields

The simplest fix involves just one character: `?`. This indicates that `secondfield` is optional.

```cue
import "github.com/grafana/scuemata"

lin: scuemata.#Lineage 
lin: Name: "OurObj"
lin: Seqs: [
    {
        schemas: [
            { // 0.0
                firstfield: string
            },
            { // 0.1
                firstfield: string
                secondfield?: int
            }
        ]
    }
]
```

Marking added fields as optional is often a good option. But it's not without implications: by making `secondfield` optional, we have nontrivially expanded the set of valid values to which consumers of an `OurObj` must assign semantics/program behavior. Before, it was "all valid values of `int`". Now, it's that, plus "an absence of a value."

If the behavior of a program accepting `OurObj` would behave in a meaningfully different way on the absence of this value vs. any valid `int`, this may be a good outcome. But if the program ends up treating the _absence_ of an `OurObj.secondfield` as equivalent to the _presence_ of some particular `int` value, the next option is likely a preferable: setting a default.

## Setting Defaults

CUE allows [specifying default values](https://cuelang.org/docs/tutorials/tour/types/defaults/) for fields. We can do so with `secondfield`, which will make `0.1` backwards compatible with `0.0`, making our lineage valid. Here, we specify `42` as the default value.

```cue
import "github.com/grafana/scuemata"

lin: scuemata.#Lineage 
lin: Name: "OurObj"
lin: Seqs: [
    {
        schemas: [
            { // 0.0
                firstfield: string
            },
            { // 0.1
                firstfield: string
                secondfield: int | *42
            }
        ]
    }
]
```

Choosing between optional and default values for added fields is a subtle, surprisingly hard problem that lacks a single, universally right answer. (In most schema-writing cases, using both at once is discouraged, as it just muddles the issue further.) In general, prefer: 

* Optional fields when the absence of said field most naturally corresponds to an absence of a class of behavior in the consuming program
* Default values when the absence of said field most naturally corresponds to the behavior when a particular value is present

Other considerations to be aware of:

* Optional values may not translate well into the language you want to work in. TypeScript has a direct symbolic equivalent, but in Go, the only sane representation is to represent the field as a pointer type and treat `nil` as absence.
* Once a default value is published, changing it is always considered a breaking change, requiring the creation of a new lineage (up next!)

## Safe Breaking Changes

The third option for making our lineage valid is to rely on scuemata's foundational feature: making breaking changes safely. In this approach, rather than adding `secondfield` to a schema the same sequence, we place our schema in a new sequence, thereby granting it the version number `1.0`.

```cue
import "github.com/grafana/scuemata"

lin: scuemata.#Lineage 
lin: Name: "OurObj"
lin: Seqs: [
    {
        schemas: [
            { // 0.0
                firstfield: string
            },
        ]
    },
    {
        schemas: [
            { // 1.0
                firstfield: string
                secondfield: int
            }
        ]
    }
]
```

Whereas successive schema in the same sequence must be backwards compatible, a successor schema in a _different_ sequence must be backwards **_in_**compatible. (The logical inverse holds.) Therefore, making `secondfield` here either optional or giving it a default will make this lineage invalid.

But this lineage is already invalid, because all sequences after the first requires the author to _also_ define a Lens.

## Defining a Lens

Lenses define a bidirectional mapping back and forth between sequences, logically connecting the final schema in one sequence to the first schema in its successor. They're the magical pixie dust that provide scuemata's foundational guarantee - translatability of instance

But - as in all software - where there be magic, [there be dragons](https://en.wikipedia.org/wiki/Here_be_dragons). Checking syntactic, type-level backwards compatibility is trivial thanks to CUE, but _semantics_, or behavior of the programs consuming these schema, are the only reasonable motivation for making a breaking change. Because [no general algorithm can exist for specifying semantic correctness](https://en.wikipedia.org/wiki/Rice%27s_theorem), it's the responsibility of lineage authors to ensure that their lenses capture semantics correctly. The only help scuemata, or any generic system, can provide is linting, e.g. "field `x` isn't mapped to the new sequence - did you mean to do that?"


## Emitting a Lacuna