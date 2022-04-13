# Invariants

TODO TODO TODO

this doc will contain two tables:

* One describing the invariants that define lineage validity, and are enforced by different parts of the toolchain
* Another describing publish-time constraints, which are largely about immutability

At least the first table will indicate the maturity of the enforcement mechanism, as well as where enforcement currently resides (native, portable CUE vs. language-specific helper libs)

Also, this is where the maths discussion should probably live that explains Thema is all a category theoretic construct

What are the axioms of the category Lineage?

## Go Assignability

**Stability note: this definition is rudimentary, and may change.**

Thema defines a relationship between its schemas and Go types based on the notion of "assignability": for all valid instances of a schema, will it be possible to accurately represent that instance in a given Go type? If so, that schema is considered to be "assignable" to that Go type.

Go Assignability is a case-specific definition of the more general [CUE subsumption relation](https://cuelang.org/docs/references/spec/#values-1), amounting to mutual subsumption (`cue ⊑ go` and `go ⊑ cue`), with certain rules relaxed when necessitated by differences in the type systems. Whether the assignable relation holds for a given Thema schema and Go type can be checked with the [`AssignableTo()` Go function](https://pkg.go.dev/github.com/grafana/thema#AssignableTo).

### Struct rules

* CUE struct types must correspond to Go struct types, named or unnamed.
* Excess fields must not be present on either side. ([Closed struct semantics](https://cuelang.org/docs/references/spec/#closed-structs) are always applied.)
* If a CUE struct field is optional (`?`), there must exist a corresponding Go struct field.

### List rules

* CUE closed list types must correspond to Go fixed array types.
* CUE open list types must correspond to Go slice types.
* CUE list types must contain at most one type constraint; Go cannot express multi-typed lists.

### Basic type rules

* Go `interface{}` or `any` (Go 1.18+) types are escape hatches, allowing any CUE type.
* CUE values having more than one basic kind (e.g. `(string|int)` may only correspond to `interface{}`/`any`.
* CUE `string` kinded-values must have corresponding Go `string` types.
* CUE `bool` kinded-values must have corresponding Go `bool` types.
* CUE `int` kinded-values must have a corresponding Go integer type must allow at least all the integral values allowed by the CUE type.
  * A CUE `int` is much larger than a Go `int`. Its corresponding Go type is `math/big.Int`.
  * `int32` and `uint32` are recommended for use in CUE schemas where use of Go's ergonomic, arch-dependent `int` and `uint` are desirable in the corresponding Go type.
* CUE `float` kinded-values must have corresponding Go `float64` types.
* CUE `number` kinded-values are not permitted. (Use `int` or `float`.)
* CUE `null` kinded-values are not permitted. (Represent optionality with `?`)

### Other rules

* Go channel, complex, and function types are not permitted.

TODO Go pointers, uints, runes, smaller number sizes, CUE & Go embeds, CUE references, improve optionality, nullability, disjunctions