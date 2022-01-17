# Tutorial: Writing a Lineage

The most fundamental task in thema is writing a lineage. Writing a lineage is similar to other schema-writing systems: you're defining a specification for what data is supposed to look like, with two key differences:

1. CUE is the only language supported for schemas
2. Schemas are written _within_ a larger, well-defined structure - the lineage - which groups those schema together and enforces certain requirements.

This tutorial focuses on the second key difference: expressing collections of schemas as a valid lineage.

**Note:** A primer on writing CUE is available using the [official CUE tutorials](https://cuelang.org/docs/tutorials/) and [CUE Playground](https://tip.cuelang.org/play/?id=#cue@export@cue). 

## Before you begin

Read the [Thema overview](overview.md) before beginning this tutorial.

**NOTE: lineage validity is not yet [fully enforced](invariants.md) in pure CUE. Consequently, lineages described as "invalid" by this tutorial may not complain when you `cue eval` them.**

## Scaffolding

Thema lineages require only two fields to be explicitly defined: 

1. `name`: the identifier for the thing schematized by the lineage. Use a simple name, not a fully-qualified name - you'll define the fully-qualified name later. For this tutorial, we'll use [`"Ship"`](https://github.com/grafana/thema/blob/main/FAQ.md#where-does-the-name-thema-come-from).
2. `seqs`: contains the list of all sequences of all schemas within the lineage, and the lenses that map between them. It's basically a two-dimensional array.

```cue
import "github.com/grafana/thema"

lin: thema.#Lineage 
lin: name: "Ship"
lin: seqs: [
    {
        schemas: []
    }
]
```

A valid lineage requires at least one sequence in `seqs`, which in turn must contain at least one schema in its `schemas` list. As a result, our example isn't actually a valid lineage. Rather, it's the minimum necessary structure to begin defining a lineage. (Note that the outermost `lin` can be omitted, in which case the entire file is the lineage.)

Supporting tooling must refuse to work with invalid lineages, similar to a failed type check. As a result, attempting to `cue eval`, [load the lineage for use in Go](https://pkg.go.dev/github.com/grafana/thema#BindLineage), or otherwise doing anything with the example here, ([should](TODOlinktoissue)) fail with a message that the length of `schemas` is less than one.

## Defining a schema

Each element in the `schemas` list is treated as a single schema. By default, these schema definitions can be any CUE value, from a simple primitive `bool` to a highly complex, deeply nested object. Complex CUE constraints may also be used, though [this may limit translatability](TODO).

Lineages should be sealed structures, as thema invariants only hold in the context of immutable schema declarations. Referencing CUE values from externally imported CUE modules within thema schema can undermine this, as those values may change when the version of the imported module is changed.

Let's define our first schema as an object containing a single field named `firstfield`, which must be of type `string`.

```cue
import "github.com/grafana/thema"

lin: thema.#Lineage 
lin: name: "Ship"
lin: seqs: [
    {
        schemas: [
            {
                firstfield: string
            }
        ]
    }
]
```

And that's it! Congratulations, we now have a valid thema lineage, containing a single schema.

## Adding more schemas

The schema we wrote isn't terribly exciting. But we can add to it.

When writing real thema, whether to define a new schema or make additions directly to the existing one is determined by whether the latest existing schema has been published, as published schema must be immutable. That makes the definition of "published" quite important.

For this tutorial, we'll sidestep the issue by assuming that publication has happened. Because publication has happened - making the schema immutable - making changes requires creating a new schema. 

Let's add one more field, `secondfield`, which must be an `int`.

**Note:** in thema, schema version is determined by its position within the two-dimensional array structure of `seqs`, rather than through arbitrary choice by the author. These structurally-determined version numbers are indicated as comments.

```cue
import "github.com/grafana/thema"

lin: thema.#Lineage 
lin: name: "Ship"
lin: seqs: [
    {
        schemas: [
            { // 0.0
                firstfield: string
            },
            { // 0.1
                firstfield: string // You can manually recreate the prior schema...
                schemas[0] // ...or just embed a reference to it.
                secondfield: int
            }
        ]
    }
]
```

This isn't a valid lineage, though. `secondfield` is required and lacks a default, which means that valid instances of `0.0` will be invalid with respect to `0.1`. Our change is backward incompatible, which breaks the rules of thema.

There are three approaches to fixing this:
- make `secondfield` optional
- give `secondfield` a default[^defaultcompat]
- start a new sequence

## Optional fields

The simplest fix involves just one character: `?`. This indicates that `secondfield` is optional.

```cue
import "github.com/grafana/thema"

lin: thema.#Lineage 
lin: name: "Ship"
lin: seqs: [
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

Marking added fields as optional is often a good option, but it's not without implications. By making `secondfield` optional, we have nontrivially expanded the set of valid values to which consumers of `Ship` must assign semantics/program behavior. Before, it was "all valid values of `int`". Now, it's that, plus "an absence of a value."

If the behavior of a program accepting `Ship` behaves in a meaningfully different way absent this value rather than any valid `int`, this outcome may be acceptable. But if the program treats the _absence_ of a `Ship.secondfield` as equivalent to the _presence_ of some particular `int` value, it's better to set a default.

## Setting defaults

CUE allows [specifying default values](https://cuelang.org/docs/tutorials/tour/types/defaults/) for fields. We can do so with `secondfield`, which will (may[^defaultcompat]) make `0.1` backward compatible with `0.0`, making our lineage valid. In this example, specify `42` as the default value.

```cue
import "github.com/grafana/thema"

lin: thema.#Lineage 
lin: name: "Ship"
lin: seqs: [
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

Choosing between optional and default values for added fields is a subtle, surprisingly hard problem that lacks a single, universally correct answer. (In most schema-writing cases, using both at once is discouraged, as it just muddles the issue further.) In general, prefer: 

* Optional fields when the absence of said field most naturally corresponds to an absence of a class of behavior in the consuming program
* Default values when the absence of said field most naturally corresponds to the behavior when a particular value is present

Other considerations:

* Optional values may not translate well into the language you want to work in. TypeScript has a direct symbolic equivalent, but in Go, the only sane representation is to represent the field as a pointer type and treat `nil` as absence.
* Once a default value is published, changing it is always considered a breaking change, requiring the creation of a new lineage 

## Safe breaking changes

The third option for making our lineage valid is to rely on thema's foundational feature: making breaking changes safely. We choose this path over the others for some reason that's important to the semantics of a newer version of our program - the field is indeed required for all future correct behavior, and relies on information - say, some user input - that simply wasn't a part of the initial schema. Requirements evolve, and programs naturally evolve them. 

In this approach, rather than adding `secondfield` to a schema in the same sequence, we place our schema in a new sequence, thereby granting the new schema the version number `1.0`.

```cue
import "github.com/grafana/thema"

lin: thema.#Lineage 
lin: name: "Ship"
lin: seqs: [
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

Whereas successive schema in the same sequence must be backward compatible, a successor schema in a _different_ sequence must be backward _in_compatible_. (The logical inverse holds.) As a result, making `secondfield` here either optional or giving it a default will make this lineage invalid.

But this lineage is already invalid, because all sequences after the first requires the author to _also_ define a Lens.

## Defining a lens

Lenses define a bidirectional mapping back and forth between sequences, logically connecting the final schema in one sequence to the first schema in its successor sequence. They're the magical connectivity that provide thema's foundational guarantee - translatability of a valid instance of a schema to other versions of that schema.

But - as in all software - where there be magic, [there be dragons](https://en.wikipedia.org/wiki/Here_be_dragons). Checking syntactic, type-level backward compatibility is trivial thanks to CUE, but _semantics_, best thought of as the intended behavior of the programs consuming schema instances, are the only reasonable motivation for making a breaking change. Because [no general algorithm can exist for specifying semantic correctness](https://en.wikipedia.org/wiki/Rice%27s_theorem), it's the responsibility of lineage authors to ensure that their lenses capture semantics correctly.

The only help thema, or any generic system, can provide is linting; for example, "field `x` isn't mapped to the new sequence - did you mean to do that?"

TODO move ^ to concepts/termdef section on lenses

Lenses map to the new sequence (`forward`) and back (`reverse`). In both directions, there's a schema being mapped `from` and `to`, and the actual mapping is encapsulated within the `rel` field.

The change to the `Ship` schema is trivial, but presents an interesting challenge - because we specifically don't want to make `secondfield` optional or give it a default value, how can we define a `rel` that still produces a valid instance of `Ship@1.0` on the other side of the `forward` mapping? (Guaranteed valid concrete lens output [is a property we hope to generically enforce, but don't yet](TODOlinktoissue).)

The only real answer is to add a placeholder value - here, `-1`.

```cue
import "github.com/grafana/thema"

lin: thema.#Lineage
lin: name: "Ship"
lin: seqs: [
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

        lens: forward: {
            from: seqs[0].schemas[0]
            to: seqs[1].schemas[0]
            rel: {
                // Direct mapping of the first field
                firstfield: from.firstfield
                // Just some placeholder int, so we have a valid instance of schema 1.0
                secondfield: -1
            }
            translated: to & rel
        }
        lens: reverse: {
            from: seqs[1].schemas[0]
            to: seqs[0].schemas[0]
            rel: {
                // Map the first field back
                firstfield: from.firstfield
            }
            translated: to & rel
        }
    }
]
```

Applied to some concrete JSON (with a `version` field implicitly added to the schema), this lens would produce the following:

```json
{
    "input": {
        "version": [0, 0],
        "firstfield": "foobar",
    },
    "output": {
        "version": [1, 0],
        "firstfield": "foobar",
        "secondfield": -1
    }
}
```

The output is valid, but less than ideal. Are we just going to have `-1` values littered all over our instances of `Ship.secondfield`? When would those get cleaned up? Does choosing `-1` as a placeholder grant special semantics to that particular value in perpetuity?

These questions bring us to the last part of thema: Lacunas.

## Emitting a lacuna

Thema's professed guarantee - all prior valid instances of schema will be translatable to all future valid instances of schema - sounds lovely. But the `secondfield` case shows it to be a pressure vessel, fit to burst when requirements evolve in just _slightly_ the wrong way. Other schema systems are similar - they "burst" when folks make breaking changes to attain the semantics they want. And it's nice that thema pushes this out further with the ability to encode translations in lenses. But eventually, it'll still burst, and folks will pick their desired semantics over thema's rules - just like they do today.

To prevent this outcome, what we really need is a pressure release valve. Which is where lacunas come in.

Lacunas represent a gap or flaw in a lens translation. As a lineage author, you add a lacuna to your lens when the translation results in a message that, while syntactically valid (it conforms to schema), has problematic semantics. Lacunas are accumulated during translation, and returned alongside the translated instance itself.

Thema defines a limited set of lacuna types that correspond to different types of flaws. (This area is under active development.) For our case, we should emit a `Placeholder` lacuna.

```cue
import "github.com/grafana/thema"

lin: thema.#Lineage
lin: name: "Ship"
lin: seqs: [
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

        lens: forward: {
            from: seqs[0].schemas[0]
            to: seqs[1].schemas[0]
            rel: {
                firstfield: from.firstfield
                secondfield: -1
            }
            lacunas: [
                thema.#Lacuna & {
                    targetFields: [{
                        path: "secondfield"
                        value: to.secondfield
                    }]
                    message: "-1 used as a placeholder value - replace with a real value before persisting!"
                    type: thema.#LacunaTypes.Placeholder
                }
            ]
            translated: to & rel
        }
        lens: reverse: {
            from: seqs[1].schemas[0]
            to: seqs[0].schemas[0]
            rel: {
                // Map the first field back
                firstfield: from.firstfield
            }
            translated: to & rel
        }
    }
]
```

Basic inputs and outputs:

```json
{
    "input": {
        "version": [0, 0],
        "firstfield": "foobar",
    },
    "output": {
        "instance": {
            "version": [1, 0],
            "firstfield": "foobar",
            "secondfield": -1
        },
        "lacunas": [
            {
                "sourceFields": [],
                "targetFields": [
                    {
                        "path": "secondfield",
                        "value": -1
                    }
                ],
                "message": "-1 used as a placeholder value - replace with a real value before persisting!",
                "type": "Placeholder"
            }
        ]
    }
}
```

Encapsulating translation flaws as lacunas relieves pressure on the schemas and translation. Schemas need not carry extraneous, legacy fields to reflect translation flaws, and lacunas can disambiguate for the calling program between translations with flaws, and those without. In this case, we can imagine `secondfield` is actually a serial identifier/foreign key, and the calling program can be constructed to look for a `Placeholder` lacuna on `secondfield`, then replace that `-1` with a correct value derived from another source.

Knowing when to emit a lacuna, and which type to emit, is nontrivial. The set of lacuna types and precise rules for when and how to use them appropriately are under active development. We will, in future, provide documentation specific to each lacuna type. In the meantime, the [exemplars directory](https://github.com/grafana/thema/tree/main/exemplars) contains a number of examples of lacuna use.

## Advanced: schema openness

TODO

## Advanced: constraining with `joinSchema`

TODO

## Wrap up

You've now seen all the component parts of thema and how they work together, which provides a sense of how to express schemas using thema. In the [next tutorial](go-mapping.md), we'll map the `Ship` lineage created here to a general-purpose language (Go), which is prerequisite to writing useful programs around lineages.

[^defaultcompat]:
    Including a default [may or may not](https://github.com/grafana/thema/discussions/15) ultimately be considered a backwards compatible change by Thema. For now, prefer using optional fields.