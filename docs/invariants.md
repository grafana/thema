# Invariants

TODO TODO TODO

this doc will contain two tables:

* One describing the invariants that define lineage validaity, and are continuously enforced by 
* Another describing publish-time constraints, which are largely about immutability

At least the first table will indicate the maturity of the enforcement mechanism, as well as where enforcement currently resides (native, portable CUE vs. language-specific helper libs)

Also, this is where the maths discussion should probably live that explains Thema is all a category theoretic construct

What are the axioms of the category Lineage?

## Go Assignability

**Stability note: this definition is rudimentary, and may change.**

Thema defines a relationship between its schemas and Go types based on the notion of "assignability": for all valid instances of a schema, will it be possible to accurately represent that instance in a given Go type? If so, that schema is considered to be "assignable" to that Go type.

Go Assignability is a case-specific definition of the more general [CUE subsumption relation](https://cuelang.org/docs/references/spec/#values-1) (`cue ⊑ go`). Whether the assignable relation holds for a given Thema schema and Go type can be checked with the [`AssignableTo()` Go function](https://pkg.go.dev/github.com/grafana/thema#AssignableTo).

### Struct-kind rules

* CUE struct types MUST correspond to Go struct types, named or unnamed.
* Excess fields MUST NOT be present on either side. ([Closed struct semantics](https://cuelang.org/docs/references/spec/#closed-structs) are always applied.)
* If a CUE struct field is optional (`?`), there MUST exist a corresponding Go type field, and it MUST be marked `omitempty` in its JSON struct tag.
* If a Go field is optional (`omitempty"` JSON struct tag), it MUST correspond to an optional (`?`) CUE field.

### List-kind rules

* CUE closed list types MUST correspond to Go fixed array types.
* CUE open list types MUST correspond to Go slice types.
* CUE list types MUST contain at most one type constraint; Go cannot express multi-typed lists.

### Basic type rules

* CUE values having more than one basic kind (e.g. `(string|int)` are not permitted.
* CUE `string` kinded-values MUST have corresponding Go `string` types.
* CUE `bool` kinded-values MUST have corresponding Go `bool` types.
* CUE `int` kinded-values MUST have corresponding Go `int64` types.
* CUE `float` kinded-values MUST have corresponding Go `float64` types.
* CUE `number` kinded-values are not permitted. (Use `int` or `float`.)
* CUE `null` kinded-values are not permitted. (Represent optionality with `?`)

TODO Go pointers, uints, smaller number sizes, CUE & Go embeds, CUE references, improve optionality, nullability, disjunctions