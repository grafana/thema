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

## Is Scuemata as expressive as other schema systems?

Scuemata is just a thin layer of naming patterns and constraints atop of CUE itself, which makes this largely a question about CUE's expressiveness.

For the most part, yes, CUE is comparably expressive to other common schema systems, like JSON Schema and OpenAPI. There are some areas where CUE is less expressive, and some where it's more.

## What definition of "backwards compatibility" does Scuemata use in its checks?

[CUE's definition of subsumption](https://cuelang.org/docs/concepts/logic): does `A` subsume `B`? If so, then `A` is backwards compatible with `B`.

This definition is precise, and a design premise of scuemata is that, because scuemata should make it easy to  precision in this definition is more important than permissiveness, as scuemata is supposed to make it easy to allow breaking changes.

## Aren't breaking changes evil? Isn't scuemata encouraging bad behavior?

If you are committed to believing this, we cannot offer definitive, contradictory proof.

Our foundational belief is that, while breaking changes can cause considerable pain, that pain has not been, and is unlikely to ever be, sufficient basis for system authors to stop making breaking changes.

Given this premise, the best course of action is to create patterns that allow breaking changes made by schema authors to be effectively managed by schema consumers. Scuemata is the simplest such pattern we can imagine: it turns "breaking" changes from hard, brittle failures into softer questions of risk management.

## Why did Scuemata make up a new version numbering system instead of just using [semver](https://semver.org)?

In scuemata, unlike most version numbering systems, a schema's version is not an arbitrary number declared by the lineage's author. Rather, version numbers are derived from the position of the schema within the lineage's list of sequences.

Sequence position, in turn, is governed by scuemata's constraints on backwards compatibility and lens existence. By tying version numbers to these checkable invariants, scuemata versions gain a precise semantics absent from systems like semver.

## How do I express prerelease-type concepts: "alpha", "beta", etc.?

You don't. 

Semantic versioning [explicitly](https://semver.org/#spec-item-9) grants prereleases an exception to its compatibility semantics. This makes each [contiguous series of] prerelease a subsequence where "anything goes."

Scuemata takes the stance that it is preferable to _never_ suspend version number-implied guarantees, and instead lean hard into the system of lenses, translations, and lacunae. In other words, it's fine to experiment and make breaking changes within your scuemata, so long as you write lenses and lacunae that can lead your users' objects to solid ground.

Support for indicating a maturity level on individual schema may be added in the future. But they would have no bearing on core scuemata invariants. Instead, a purely opaque signal between humans: "we're really not sure about this yet; future lenses for translating from this schema may be sloppy!"
