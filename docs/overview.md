# About Thema

Thema is a system for expressing schema for a particular kind of object and safely evolving that schema definition over time, including across what would otherwise be breaking changes. Thema consists of:

* **Components:** schemas themselves, and related logical structures around those schemas. Components are the "nouns" of the system, roughly analogous to classes or types.
* **Operations:** things that can be done with valid components. These are the "verbs" of the system, roughly analogous to functions or methods.
* **Guarantees:** whole-system properties that arise from machine-checked constraints on what it means to be a valid component. These are the universal rules of Thema, roughly analogous to type checking or rules of syntax.

Thema schemas (and the other components) are written in CUE, much like implementing an interface. Operations are how you build useful programs based on Thema. They may be performed directly in CUE, or from any language with a [CUE spec](https://cuelang.org/docs/references/spec)-compliant evaluator, which is necessary for writing language bindings - currently only Go.

Invariants are ideally expressed and checked as pure CUE constraints, resulting in uniform guarantees across the consuming language. That's a [WIP](invariants.md), and for now some must be checked in the consuming language (so, Go).

Components and operations are exposed as exported structs definitions in CUE (no standard docs system yet) and as a [library in Go](https://pkg.go.dev/github.com/grafana/thema).

## About Thema Components

Thema is built from six fundamental logical components:

* **Schema:** The common meaning - a specification for the shape of some data.
* **Sequence:** An ordered list of schema, each backward compatible with its predecessors.
* **Lens:** A bidirectional mapping between the final schema in one sequence, and the first schema in another sequence.
* **Lineage:** An ordered list of sequences, linked tail-to-head by lenses, that collectively contain the history of all schemas for a single kind of object.
* **Instance:** Some data that is valid with respect to a schema.
* **Lacuna:** A gap in a lens's mapping logic such that a particular instance, having passed through that lens, has some meaningful semantic deficiency.

These logical components are deeply interlinked as illustrated below. The first four components - schema, sequence, lens, and lineage - can be understood independent of a program that is running a useful operation. If we had a valid lineage containing two sequences, each containing two schema, and thus necessarily one explicitly-defined lens, it would look like this:

![Abstract Lineage](lineage-structure.png)

Lenses within a sequence are implicit, because that's precisely what backward compatibility means: instances of old schema remain valid with respect to newer schema, without the need for translation. (NOTE: diagram notwithstanding, this is clearly not true in reverse - this is a [TODO](https://github.com/grafana/thema/issues/6).)

The remaining two concepts - instance and lacuna - are most easily understood in the context of a program performing operations on a lineage. Beyond that, it's useful to review concrete examples, either via the [Lineage authoring tutorial](authoring.md) or by looking at the [exemplars](https://github.com/grafana/thema/tree/main/exemplars).

## About Thema Operations

Thema operations allow programs to combine data with a lineage and its schema. Because lineages are collections of schema, programs must first decide which schema to use. Two key operations assist with selecting an individual schema out of the lineage:

* **`Schema()`:** given a version number of the format major.minor, get a particular schema from a lineage.
* **`ValidateAny()`:** given some data, search the lineage for a schema that the data validates against.

In the Go library, these are the methods on the `Lineage` [interface](https://pkg.go.dev/github.com/grafana/thema#Lineage).

Individual schema, once chosen, have one key operation:

* **`Validate()`:** given a schema from a lineage, check whether some data is valid instance of that schema

In Thema's Go library, successful validation of some data returns an [`Instance`](https://pkg.go.dev/github.com/grafana/thema#Instance), which presents the final key operation:

* **`Translate()`:** given an instance of a schema in a lineage, transform it to an instance of some other schema in the same lineage, and emit any lacuna arising from the transformation.

### Usage Patterns

There are some common usage patterns for integrating Thema into programs. Where possible, we codify these patterns into language library helpers, called "kernels".

The most common pattern (codified as [`InputKernel`](https://pkg.go.dev/github.com/grafana/thema/kernel#InputKernel) in Go) is to accept input data from any schema, then converge onto a single version of the schema. This allows your program to accept all versions of the schema for your object, but to be written against only one form.

This pattern begins with a three-step process, typically executed at the program boundary when input is first received:

1. Receive some input data and `ValidateAny()` to confirm it is an instance, and of what syntactic version of the schema
2. `Translate()` the instance to the syntactic version of the schema the program is currently designed to work with
3. Decide what to do with any lacunas emitted from translation - for example: ignore, log, error out, mutate the translated instance

This animation illustrates a program performing these first two steps across varying numbers of the schemas from the example above:

![Validate and Translate](validate-and-translate.gif) TODO fixup the graffle, make the gif

Once this process is complete, the program can continue (or terminate based on observed lacunas) to perform useful behavior based on the input, now known to be both a) valid and b) represented in the form of the syntactic version of the schema against which the program has been written. Versioning and translation has been encapsulated at the program boundary, and the rest of the program can safely pretend that only the one version of the schema exists.

Deeper exploration and concrete examples are available in the [tutorial on using Thema from Go](go-usage.md).

## About Guarantees

Thema's benefits over other schema systems comes from the guarantees it provides to code depending on a Thema lineage. "Depending" here includes code in the same repository, written by the lineage's author. In sum, the guarantee is:

**You can write your program against any schema in a lineage, and know that any input valid against any schema in that lineage will be translatable to the schema your program expects.**

Coordination is the Achilles heel of all distributed systems. Thema's guarantee eliminates the need to coordinate the deployment of independent systems that pass messages to each other. The communication contract between these systems [is no longer individual schema, but the lineage](https://github.com/grafana/thema/blob/main/FAQ.md#you-cant-fool-me-breaking-changes-are-breaking---how-can-they-possibly-be-made-non-breaking). This **evolutionary decoupling** allows for a novel class of fundamentally decentralized development.

Thema's guarantee arises from the combination of smaller, machine-checkable constraints on what constitutes a valid lineage. **Not all planned rules are fully implemented as checked invariants yet; until they are, this guarantee is wobbly, and the Thema project should be considered unstable.** The [invariants (TODO) doc](invariants.md) enumerates the granular rules, their completeness, and enforcement mechanism.

Thema will be considered a mature, stable project when all the intended invariants are machine-checked. Even when this milestone is reached, however, certain caveats will remain:

* Programs need to have the most updated version of the lineage, in case they receive inputs that are valid, but against a schema they are not yet aware of. This implies a publishing and distribution model for lineages is necessary, as well as an append-only immutability requirement. See [Publishing (TODO)](publishing.md).
* Even with lenses and lacunas, some data semantics will result in practical limits on surpriseless translation of message intent. As the Project Cambria folks [note](https://www.inkandswitch.com/cambria/#findings), a practical breaking point will eventually be reached. Thema is not a magical silver bullet.
