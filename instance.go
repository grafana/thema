package thema

import (
	"fmt"

	"cuelang.org/go/cue"
)

// BindInstanceType produces a TypedInstance, given an Instance and a
// TypedSchema derived from its Instance.Schema().
//
// The only possible error occurs if the TypedSchema is not derived from the
// Instance.Schema().
func BindInstanceType[T Assignee](inst *Instance, tsch TypedSchema[T]) (*TypedInstance[T], error) {
	if !schemaIs(inst.Schema(), tsch) {
		return nil, fmt.Errorf("typed schema is not derived from instance's schema")
	}

	return &TypedInstance[T]{
		inst: inst,
		tsch: tsch,
	}, nil
}

// An Instance represents some data that has been validated against a
// lineage's schema. It includes a reference to the schema.
type Instance struct {
	// The CUE representation of the input data
	raw cue.Value
	// A name for the input data, primarily for use in error messages
	name string
	// The schema the data validated against/of which the input data is a valid instance
	sch Schema
}

// Hydrate returns a copy of the Instance with all default values specified by
// the schema included.
//
// NOTE hydration implementation is a WIP. If errors are encountered, the
// original input is returned unchanged.
func (i *Instance) Hydrate() *Instance {
	i.sch.Lineage().Runtime()
	ni, err := doHydrate(i.sch.UnwrapCUE(), i.raw)
	// FIXME For now, just no-op it if we error
	if err != nil {
		return i
	}

	return &Instance{
		raw:  ni,
		name: i.name,
		sch:  i.sch,
	}
}

// Dehydrate returns a copy of the Instance with all default values specified by
// the schema removed.
//
// NOTE dehydration implementation is a WIP. If errors are encountered, the
// original input is returned unchanged.
func (i *Instance) Dehydrate() *Instance {
	ni, _, err := doDehydrate(i.sch.UnwrapCUE(), i.raw)
	// FIXME For now, just no-op it if we error
	if err != nil {
		return i
	}

	return &Instance{
		raw:  ni,
		name: i.name,
		sch:  i.sch,
	}
}

// AsSuccessor translates the instance into the form specified by the successor
// schema.
//
// TODO figure out how to represent unary vs. composite lineages here
func (i *Instance) AsSuccessor() (*Instance, TranslationLacunas) {
	return i.Translate(i.sch.Successor().Version())
}

// AsPredecessor translates the instance into the form specified by the predecessor
// schema.
//
// TODO figure out how to represent unary vs. composite lineages here
func (i *Instance) AsPredecessor() (*Instance, TranslationLacunas) {
	panic("TODO translation from newer to older schema is not yet implemented")
}

// UnwrapCUE returns the cue.Value representing the instance's underlying data.
func (i *Instance) UnwrapCUE() cue.Value {
	return i.raw
}

// Schema returns the schema which subsumes/validated this instance.
func (i *Instance) Schema() Schema {
	return i.sch
}

func (i *Instance) rt() *Runtime {
	return getLinLib(i.Schema().Lineage())
}

type TypedInstance[T Assignee] struct {
	inst *Instance
	tsch TypedSchema[T]
}

func (inst *TypedInstance[T]) TypedSchema() TypedSchema[T] {
	return inst.tsch
}

func (inst *TypedInstance[T]) Value() (T, error) {
	t := inst.tsch.NewT()
	// TODO figure out correct pointer handling here
	err := inst.inst.raw.Decode(&t)
	return t, err
}

func (inst *TypedInstance[T]) ValueP() T {
	t, err := inst.Value()
	if err != nil {
		panic(fmt.Errorf("error decoding value: %w", err))
	}
	return t
}

// Translate transforms the Instance so that it's an instance of another schema
// in the lineage. A new *Instance is returned representing the transformed
// value, along with any lacunas accumulated along the way.
//
// Forward translation within a sequence (e.g. 0.0 to 0.7) is trivial, as
// all those schema changes are established as backwards compatible by Thema's
// lineage invariants. In such cases, the lens is referred to as implicit, as
// the lineage author does not write it, with translation relying on simple
// unification. Lacunas cannot be emitted from such translations.
//
// Forward translation across sequences (e.g. 0.0 to 1.0), and all reverse
// translation regardless of sequence boundaries (e.g. 1.1 to either 1.0
// or 0.0), is nontrivial and relies on explicitly defined lenses, which
// introduce room for lacunas and author judgment.
//
// Thema translation is non-invertible over instances in the general case by
// design. That is, Thema does not guarantee that translating an instance from
// 0.0->1.0->0.0 will result in the exact original instance. (Input state
// preservation can be fully achieved in a wrapping layer, so we avoid introducing
// complexity into Thema that is not essential for all use cases.)
//
// NOTE reverse translation is not yet supported, and attempting it will panic.
//
// TODO define this in terms of AsSuccessor and AsPredecessor, rather than those in terms of this.
func (i *Instance) Translate(to SyntacticVersion) (*Instance, TranslationLacunas) {
	if to.Less(i.Schema().Version()) {
		panic(fmt.Sprintf("FIXME translation of instances from newer to older schema is not yet implemented - %s->%s was requested", i.Schema().Version(), to))
	}
	newsch, err := i.Schema().Lineage().Schema(to)
	if err != nil {
		panic(fmt.Sprintf("no schema in lineage with version %v, cannot translate", to))
	}

	out, err := cueArgs{
		"linst": i.asLinkedInstance(),
		"to":    to,
	}.call("#Translate", i.rt())
	if err != nil {
		// This can't happen without a name change or an invariant violation
		panic(err)
	}

	lac := make(multiTranslationLacunas, 0)
	out.LookupPath(cue.MakePath(cue.Str("lacunas"))).Decode(&lac)

	return &Instance{
		raw:  out.LookupPath(cue.MakePath(cue.Str("linst"), cue.Str("inst"))),
		name: i.name,
		sch:  newsch,
	}, lac
}

// TODO generic-typed Translation

type multiTranslationLacunas []struct {
	V   SyntacticVersion `json:"v"`
	Lac []Lacuna         `json:"lacunas"`
}

func (lac multiTranslationLacunas) AsList() []Lacuna {
	// FIXME This loses info, naturally - need to rework the lacuna types
	var l []Lacuna
	for _, v := range lac {
		l = append(l, v.Lac...)
	}
	return l
}

// func TranslateComposed(lin ComposedLineage) {

// }

func (i *Instance) asLinkedInstance() cue.Value {
	li, err := cueArgs{
		"inst": i.raw,
		"lin":  i.Schema().Lineage().UnwrapCUE(),
		"v":    i.Schema().Version(),
	}.make("#LinkedInstance", i.rt())
	if err != nil {
		// This can't happen without a name change or an invariant violation
		panic(err)
	}

	return li
}
