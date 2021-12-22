# Thema Overview

Thema is a system for expressing schema for a particular kind of object and safely evolving that schema definition over time, including across what would otherwise be breaking changes. Thema consists of three kinds of things:

* **Components:** named, related logical structures that form the foundation of the system. (rough analogy: classes, nouns)
* **Operations:** things that can be done, given valid components. (rough analogy: functions, verbs)
* **Guarantees:** essential system properties that arise from invariants ensured by constraints determining the validity of particular Thema instances (rough analogy: type checking, rules of syntax).

Thema's components are expressed in CUE, and Thema schemas must also be authored in CUE. Operations are how you build useful programs based on Thema. They may be performed directly in CUE, or from any language with a [CUE spec](https://cuelang.org/docs/references/spec)-compliant evaluator, which is necessary for writing language bindings - currently only Go.

Invariants are ideally expressed and checked as pure CUE constraints, resulting in uniform guarantees across the consuming language. That's a [WIP](https://github.com/grafana/thema/issues/6), and for now some must be checked in the consuming language (so, Go).

Components and operations are exposed as exported structs definitions in CUE (no standard docs system yet) and as a [library in Go](https://pkg.go.dev/github.com/grafana/thema).

## Components

Thema is built from six fundamental logical components:

* **Schema:** The common meaning - a specification for the shape of some data.
* **Sequence:** An ordered list of Schema, each being backwards compatible with its predecessors.
* **Lens:** A bidirectional mapping between the final Schema in one Sequence, and the first Schema in another Sequence.
* **Lineage:** An ordered list of Sequences, linked tail-to-head by Lenses, that collectively contain the history of all Schemas for a single kind of object.
* **Instance:** Some data that is valid with respect to a Schema.
* **Lacuna:** A gap in a Lens's mapping logic such that a particular Instance, having passed through that Lens, has some meaningfully semantic deficiency.

_(Names are capitalized for referential emphasis in this overview doc; capitalization is not the norm elsewhere.)_

These pieces are deeply interlinked. Fortunately, we can express the web of relations visually. The first four components - Schema, Sequence, Lens, and Lineage - are easy to think about independent of any particular program trying to do some useful operation. If we had a valid Lineage containing two Sequences, each containing two Schema, and thus necessarily one explicitly-defined Lens, we could visualize it as:

![Abstract Lineage](lineage-structure.png)

Lenses within a Sequence are implicit, because that's precisely what backwards compatibility means: Instances of old Schema are still valid with respect to newer Schema, without the need for translation. (NOTE: diagram notwithstanding, this is clearly not true in reverse - this is a [TODO](https://github.com/grafana/thema/issues/6).)

The remaining two concepts - Instance and Lacuna - are best illustrated in the context of a program performing operations on a Lineage.

For more on components, it's best to see concrete examples, either via the [Lineage authoring tutorial](authoring.md) or by looking at the [exemplars](https://github.com/grafana/thema/tree/main/exemplars).

## Operations

Thema has three essential operations, given a valid Lineage:

* **Pick():** get a particular Schema from a Lineage by version number.
* **Validate():** check whether some data is an Instance - whether it's valid with respect to a particular Schema.
* **Translate():** given an Instance of some Schema in a Lineage, transform it to an Instance of some other Schema in the same Lineage, and emit any Lacuna arising from the transformation.

Other helpers exist, but are composed from these parts. Most programs that work with Thema will base most of their operations around `Validate()` and `Translate()` in a three-step process, typically executed at the program boundary when input is first received:

1. `Validate()` some data
2. `Translate()` it to the Schema version the program is currently designed to work with
3. Decide what to do (e.g. ignore, log, error out, mutate the transformed Instance) with any emitted Lacunae

Once these steps are complete, the program can continue on (or terminate based on obseved Lacunae) to perform useful behavior based on the input, which is now both a) valid and b) represented in the form of the Schema version against which the program has been written. 

This animation illustrates a program performing these first two steps across varying numbers of the Schemas from our example above:

![Validate and Translate](validate-and-translate.gif) TODO fixup the graffle, make the gif

These operations are exposed directly in CUE, and should be present in any consuming language library.

### Defaults and Encoding

There are two other basic operations, as well, though they deal with encoding CUE in other representations (e.g. JSON). They are less foundational, and only exposed in a Thema-consuming language library:

* **Dehydrate():** shrink an Instance by removing all fields containing values redundant with/equal to Schema-specified [defaults](https://cuelang.org/docs/tutorials/tour/types/defaults/).
* **Hydrate():** fill an Instance's absent fields with their Schema-defined defaults. 

Defaults are a deceptively subtle and complex topic, and are explored further [in a separate (TODO) doc](defaults.md).

## Guarantees

The fundamental reason to use Thema rather than some other schema system is the guarantee it provides to code depending on a Lineage. "Dependers" here very much includes code in the same repository, written by the Lineage's author. Simply put, that guarantee is:

**You can write your program against any Schema in a Lineage, and know that any input valid against any Schema in that Lineage will be translatable to the Schema your program expects.**

This guarantee eliminates the need to coordinate - the Achilles heel of all distributed systems - the deployment of independent systems that pass messages to each other, even indirectly. The communication contract between these systems [is no longer individual schema, but the Lineage](https://github.com/grafana/thema/blob/main/FAQ.md#you-cant-fool-me-breaking-changes-are-breaking---how-can-they-possibly-be-made-non-breaking). This **evolutionary decoupling** allows for a novel class of fundamentally decentralized development.

Thema's guarantee arises from the combination of smaller, machine-checkable constraints on what constitutes a valid Lineage. **Not all planned rules are fully implemented as checked invariants yet; until they are, this guarantee is wobbly, and the Thema project should be considered unstable.** The [Invariants (TODO) doc](invariants.md) enumerates the granular rules, their completeness, and enforcement mechanism.

Thema itself will be considered a mature, stable project when all the intended invariants are machine-checked. Even when this milestone is reached, however, certain caveats will remain:

* Programs need to have the most updated version of the Lineage, lest they receive inputs that are valid, but against a Schema they are not yet aware of. This implies a publishing and distribution model for Lineages is necessary, as well as an append-only immutability requirement. See [Publishing (TODO)](publishing.md).
* Even with Lenses and Lacunae, some data semantics will result in practical limits on surpriseless translation of message intent. As the Project Cambria folks [note](https://www.inkandswitch.com/cambria/#findings), a practical breaking point will eventually be reached.
