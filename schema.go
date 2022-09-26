package thema

import "cuelang.org/go/cue"

var (
	_ Schema                = &UnarySchema{}
	_ TypedSchema[Assignee] = &unaryTypedSchema[Assignee]{}
)

// A UnarySchema is a Go facade over a Thema schema that does not compose any
// schemas from any other lineages.
type UnarySchema struct {
	// TODO panic button if empty, nil
	raw    cue.Value
	defraw cue.Value
	lin    *UnaryLineage
	v      SyntacticVersion
}

func (sch *UnarySchema) rt() *Runtime {
	return sch.Lineage().Runtime()
}

// Validate checks that the provided data is valid with respect to the
// schema. If valid, the data is wrapped in an Instance and returned.
// Otherwise, a nil Instance is returned along with an error detailing the
// validation failure.
//
// While Validate takes a cue.Value, this is only to avoid having to trigger
// the translation internally; input values must be concrete. To use
// incomplete CUE values with Thema schemas, prefer working directly in CUE,
// or if you must, rely on UnwrapCUE().
//
// TODO should this instead be interface{} (ugh ugh wish Go had discriminated unions) like FillPath?
func (sch *UnarySchema) Validate(data cue.Value) (*Instance, error) {
	sch.rt().rl()
	defer sch.rt().ru()
	// TODO which approach is actually the right one, unify or subsume? ugh
	// err := sch.raw.Subsume(data, cue.All(), cue.Raw())
	// if err != nil {
	// 	return nil, err
	// 	// return nil, mungeValidateErr(err, sch)
	// }

	x := sch.defraw.Unify(data)
	if err := x.Validate(cue.Final(), cue.All()); err != nil {
		return nil, mungeValidateErr(err, sch)
	}

	return &Instance{
		raw:  data,
		sch:  sch,
		name: "", // FIXME how are we getting this out?
	}, nil
}

// Successor returns the next schema in the lineage, or nil if it is the last schema.
func (sch *UnarySchema) Successor() Schema {
	if s := sch.successor(); s != nil {
		return s
	}
	return nil
}

func (sch *UnarySchema) successor() *UnarySchema {
	if sch.lin.allv[len(sch.lin.allv)-1] == sch.v {
		return nil
	}

	succv := sch.lin.allv[searchSynv(sch.lin.allv, sch.v)+1]
	return sch.lin.schema(succv)
}

// Predecessor returns the previous schema in the lineage, or nil if it is the first schema.
func (sch *UnarySchema) Predecessor() Schema {
	if s := sch.predecessor(); s != nil {
		return s
	}
	return nil
}

func (sch *UnarySchema) predecessor() *UnarySchema {
	if sch.v == synv() {
		return nil
	}

	predv := sch.lin.allv[searchSynv(sch.lin.allv, sch.v)-1]
	return sch.lin.schema(predv)
}

// LatestVersionInSequence returns the version number of the newest (largest) schema
// version in the provided sequence number.
//
// An error indicates the number of the provided sequence does not exist.
func (sch *UnarySchema) LatestVersionInSequence() SyntacticVersion {
	// Lineage invariants preclude an error
	sv, _ := LatestVersionInSequence(sch.lin, sch.v[0])
	return sv
}

// UnwrapCUE returns the cue.Value that represents the underlying CUE schema.
func (sch *UnarySchema) UnwrapCUE() cue.Value {
	return sch.raw
}

// Version returns the schema's version number.
func (sch *UnarySchema) Version() SyntacticVersion {
	return sch.v
}

// Lineage returns the lineage that contains this schema.
func (sch *UnarySchema) Lineage() Lineage {
	return sch.lin
}

func (sch *UnarySchema) defPathFor() cue.Path {
	return defPathFor(sch.lin.Name(), sch.v)
}

func (sch *UnarySchema) _schema() {}

// BindType produces a TypedSchema, given a Schema that is AssignableTo() the
// provided struct type. An error is returned if the provided Schema is not
// assignable to the given struct type.
func BindType[T Assignee](sch Schema, t T) (TypedSchema[T], error) {
	if err := AssignableTo(sch, t); err != nil {
		return nil, err
	}

	tsch := &unaryTypedSchema[T]{
		Schema: sch,
		new:    t, // TODO test if this works as expected on pointers
	}
	err := sch.UnwrapCUE().Decode(&tsch.new)
	if err != nil {
		return nil, err
	}

	tsch.tlin = &unaryConvLineage[T]{
		Lineage: sch.Lineage(),
		tsch:    tsch,
	}

	return tsch, nil
}

func schemaIs(s1, s2 Schema) bool {
	panic("TODO")
}

type unaryTypedSchema[T Assignee] struct {
	Schema
	new  T
	tlin ConvergentLineage[T]
}

func (sch *unaryTypedSchema[T]) New() T {
	return sch.new
}

func (sch *unaryTypedSchema[T]) ValidateTyped(data cue.Value) (*TypedInstance[T], error) {
	inst, err := sch.Schema.Validate(data)
	if err != nil {
		return nil, err
	}

	return &TypedInstance[T]{
		inst: inst,
		tsch: sch,
	}, nil
}
func (sch *unaryTypedSchema[T]) ConvergentLineage() ConvergentLineage[T] {
	return sch.tlin
}
