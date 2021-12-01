# FAQ

## Why, why, _whyyy_ does the universe need yet another schema system?

No existing schema system was reasonably capable of encapsulating all of the following within a unified, portable, verifiable structure:

1. Schemas themselves
2. Backwards compatibility guarantees
3. Logic for translating resources across schema versions

Scuemata exists because we believe that the combination of these things fundamentally changes the kinds of systems we can build.

## You can't fool me. Breaking changes are breaking - how can they possibly be made non-breaking?

That's true. A breaking change to a contract like a schema is still breaking.

What scuemata does is change the nature of the contract between communicating systems. Instead of agreeing on a single schema, they agree on the whole scuemata lineage as the contract, with all the invariants about translation between schema versions that that entails.

For example, say system `A` accepts messages which is comprised of a single field named `foo`, which has value of type `int64`. System `B` accepts the contract, and starts sending messages to `A` according to this schema. What scuemata changes is not so much the schema in use at any one time, but the contract between `A` and `B`:

* **Traditional Schema:** `A` promises that messages with field `foo` containing an `int64` value will be valid in perpetuity.
* **Scuemata:** `A` promises that messages with a field `foo` containing an `int64` will either be valid itself, or will be translatable into a valid message, in perpetuity.

Scuemata shifts the contract up a level of abstraction - from rigid adherence to the contents of an individual schema, to the meta-property of relations between schemas.

## Is scuemata as expressive as other schema systems?

Scuemata is just a thin layer of naming patterns and constraints atop of CUE itself, which makes this largely a question about CUE's expressiveness.

For the most part, yes, CUE is comparably expressive to other common schema systems, like JSON Schema and OpenAPI. There are some areas where CUE is less expressive, and some where it's more. (TODO - links to more relevant info)

## What definition of "backwards compatibility" does scuemata use in its checks?

[CUE's definition of subsumption](https://cuelang.org/docs/concepts/logic): does `A` subsume `B`? If so, then `A` is backwards compatible with `B`.

This definition is precise, and a design premise of scuemata is that, because scuemata should make it easy to  precision in this definition is more important than permissiveness, as scuemata is supposed to make it easy to allow breaking changes.

## Aren't breaking changes evil? Isn't scuemata encouraging bad behavior?

If you are committed to believing this, we cannot offer definitive, contradictory proof.

Our foundational belief is that, while breaking changes can cause considerable pain, that pain has not been, and is unlikely to ever be, sufficient basis for system authors to stop making breaking changes.

Given this premise, the best course of action is to create patterns that allow breaking changes made by schema authors to be effectively managed by schema consumers. Scuemata is the simplest such pattern we can imagine: it turns "breaking" changes from hard, brittle failures into softer questions of risk management.

## Why did scuemata make up a new version numbering system instead of just using [semver](https://semver.org)?

Scuemata schema versions, unlike most version numbering systems, are not an arbitrary declaration by the schema author. Rather, version numbers are derived from the position of the schema within the lineage's list of sequences. Sequence position, in turn, is governed by scuemata's checked invariants on backwards compatibility and lens existence.

By associating version numbers with logical properties, scuemata versions gain precise semantics absent from other numbering systems.

## How do I express prerelease-type concepts: "alpha", "beta", etc.?

You don't. 

Semantic versioning [explicitly](https://semver.org/#spec-item-9) grants prereleases an exception to its compatibility semantics. This makes each [contiguous series of] prerelease a subsequence where "anything goes."

Scuemata takes the stance that it is preferable to _never_ suspend version number-implied guarantees, and instead lean hard into the system of lenses, translations, and lacunae. In other words, it's fine to experiment and make breaking changes within your scuemata, so long as you write lenses and lacunae that can lead your users' objects to solid ground.

Support for indicating a maturity level on individual schema may be added in the future. But they would have no bearing on core scuemata invariants. Instead, a purely opaque signal between humans: "we're really not sure about this yet; future lenses for translating from this schema may be sloppy!"
