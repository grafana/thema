# Thema

Thema is a system for writing schemas. Much like JSON Schema or OpenAPI, it is general-purpose and its most obvious application is as an [Interface Definition Language](https://en.wikipedia.org/wiki/Interface_description_language). Unlike those systems, Thema focuses not just on individual schema, but on the _evolution_ of those schema over time.

An analogy is helpful. ["Branching by abstraction"](https://martinfowler.com/bliki/BranchByAbstraction.html) suggests that you refactor large applications not with long-running VCS branches and big-bang merges, but by letting old and new code live side-by-side on `main`, and choosing between them with logical gates, like [feature flags](https://featureflags.io/feature-flags/). Thema is "schema versioning by abstraction": all versions of a schema live side-by-side on `main`, within logical structures Thema defines.

This holistic view allows Thema to act like a typechecker, but for change-safety _between_ schema versions: either schema versions must be backwards compatible, or there must exist logic to translate a valid instance of schema from one schema version to the next. [CUE](https://cuelang.org), the language in which Thema schemas are written, allows Thema to [mechanically verify these properties](#Maturity).

These capabilities make Thema a general framework for decoupling the evolution of communicating systems. This can be outward-facing: Thema's guardrails allow anyone to create APIs with Stripe's renowned [backwards compatibility](https://stripe.com/docs/upgrades) guarantees. Or it can be inward-facing: or to change the messages passed in a mesh of microservices without intricately orchestrating deployment.

Learn more in our [docs](https://github.com/grafana/thema/tree/main/docs), or in this [overview video](https://www.youtube.com/watch?v=PpoS_ThntEM)! (Some things have been renamed since that video, but the logic is unchanged.)

## Maturity

Thema is a young project. The goals are large, but bounded: we will know when the core system is complete. And it mostly is, now - though some breaking changes to how schemas are written are planned before reaching stability.

It is not yet recommended to replace established, stable systems with Thema, but experimenting with doing so is reasonable (and appreciated!). For newer projects, Thema may be a good choice today; the decision is likely to come down to whether the benefit of a simpler architecture for authoring, composing and evolving schema will offset the cost of having to chase some breaking changes.