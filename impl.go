package thema

import (
	"fmt"

	"cuelang.org/go/cue"
)

type ErrValueNotExist struct {
	path string
}

func (e *ErrValueNotExist) Error() string {
	return fmt.Sprintf("value from path %q does not exist, absent values cannot be lineages", e.path)
}

// BindLineage takes a raw cue.Value, checks that it is a valid lineage (that it
// upholds the invariants which undergird Thema's translatability guarantees),
// and returns the cue.Value wrapped in a Lineage, iff validity checks succeed. The Lineage type
// provides access to all the types and functions for working with Thema in Go.
//
// This function is the sole intended mechanism for creating Lineage objects,
// thereby providing a practical promise that all instances of Lineage uphold
// Thema's invariants. It is primarily intended for use by authors of lineages
// in the creation of a LineageFactory.
func BindLineage(raw cue.Value, lib Library, opts ...BindOption) (Lineage, error) {
	// The candidate lineage must exist.
	if !raw.Exists() {
		return nil, &ErrValueNotExist{
			path: raw.Path().String(),
		}
	}

	// The candidate lineage must be error-free.
	if err := raw.Validate(cue.Concrete(false)); err != nil {
		return nil, err
	}

	// The candidate lineage must be an instance of #Lineage.
	dlin := lib.linDef()
	if err := dlin.Subsume(raw, cue.Raw(), cue.Schema()); err != nil {
		return nil, err
	}

	cfg := &bindConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	if !cfg.skipbuggychecks {
		// The sequences and schema in the candidate lineage must follow
		// backwards [in]compatibility rules.
		if err := verifySeqCompatInvariants(raw, lib); err != nil {
			return nil, err
		}
	}

	return &UnaryLineage{
		validated: true,
		raw:       raw,
		lib:       lib,
	}, nil
}

type compatInvariantError struct {
	rawlin    cue.Value
	violation [2][2]int
	detail    error
}

func (e *compatInvariantError) Error() string {
	panic("TODO")
}

// Assumes that lin has already been verified to be subsumed by #Lineage
func verifySeqCompatInvariants(lin cue.Value, lib Library) error {
	seqiter, _ := lin.LookupPath(cue.MakePath(cue.Str("Seqs"))).List()
	var seqv int
	var predecessor cue.Value
	var predv [2]int
	for seqiter.Next() {
		var schv int
		schemas := seqiter.Value().LookupPath(cue.MakePath(cue.Str("schemas")))
		schiter, _ := schemas.List()
		for schiter.Next() {
			if schv == 0 && seqv == 0 {
				// Very first schema, no predecessor to compare against
				continue
			}

			sch := schiter.Value()
			bcompat := sch.Subsume(predecessor, cue.Raw(), cue.Schema())
			if (schv == 0 && bcompat == nil) || (schv != 0 && bcompat != nil) {
				return &compatInvariantError{
					rawlin:    lin,
					violation: [2][2]int{predv, {seqv, schv}},
					detail:    bcompat,
				}
			}

			predecessor = sch
			predv = [2]int{seqv, schv}
			schv++
		}
		seqv++
	}

	return nil
}

// A UnaryLineage is a Go facade over a valid CUE lineage that does not compose
// other lineage.
type UnaryLineage struct {
	validated bool
	name      string
	first     Schema
	raw       cue.Value
	lib       Library
}

func (lin *UnaryLineage) First() Schema {
	return lin.first
}

func (lin *UnaryLineage) RawValue() cue.Value {
	return lin.raw
}

func (lin *UnaryLineage) Name() string {
	return lin.name
}

func (lin *UnaryLineage) _lineage() {}

// A UnarySchema is a Go facade over a Thema schema that does not compose any
// schemas from any other lineages.
type UnarySchema struct {
	raw        cue.Value
	pred, succ *UnarySchema
	lin        *UnaryLineage
	v          SyntacticVersion
}

func (sch *UnarySchema) Validate(data cue.Value) (*Instance, error) {
	err := sch.raw.Subsume(data, cue.Concrete(true))
	if err != nil {
		return nil, err
	}

	return &Instance{
		raw:  data,
		sch:  sch,
		name: "", // FIXME how are we getting this out?
	}, nil
}

func (sch *UnarySchema) Successor() Schema {
	return sch.succ
}

func (sch *UnarySchema) Predecessor() Schema {
	return sch.pred
}

func (sch *UnarySchema) RawValue() cue.Value {
	return sch.raw
}

func (sch *UnarySchema) Version() SyntacticVersion {
	return sch.v
}

func (sch *UnarySchema) Lineage() Lineage {
	return sch.lin
}

func (sch *UnarySchema) _schema() {}
