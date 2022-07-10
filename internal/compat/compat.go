package compat

import (
	"fmt"

	"cuelang.org/go/cue"
	"github.com/grafana/thema"
)

func CheckSeqs(raw cue.Value) error {
	seqiter, err := raw.LookupPath(cue.MakePath(cue.Str("seqs"))).List()
	if err != nil {
		return fmt.Errorf("not a list: %w", err)
	}

	var vmaj uint
	var predecessor cue.Value
	var predsv thema.SyntacticVersion
	for seqiter.Next() {
		var vminor uint
		schemas := seqiter.Value().LookupPath(cue.MakePath(cue.Str("schemas")))
		schiter, _ := schemas.List()
		for schiter.Next() {
			v := synv(vmaj, vminor)
			sch := schiter.Value()

			// No predecessor to compare against with the very first schema
			if !(vminor == 0 && vmaj == 0) {
				// TODO Marked as buggy until we figure out how to both _not_ require
				// schema to be closed in the .cue file, _and_ how to detect default changes
				// if !cfg.skipbuggychecks {
				// The sequences and schema in the candidate lineage must follow
				// backwards [in]compatibility rules.
				// TODO Subsumption may not be what we actually want to check here,
				// as it does not allow the addition of required fields with defaults
				bcompat := ThemaCompatible(predecessor, sch)
				if (vminor == 0 && bcompat == nil) || (vminor != 0 && bcompat != nil) {
					return &CompatInvariantError{
						rawlin:    raw,
						violation: [2]thema.SyntacticVersion{predsv, v},
						detail:    bcompat,
					}
				}
				// }
			}

			predecessor = sch
			predsv = v
			vminor++
		}
		vmaj++
	}
	return nil
}

func ThemaCompatible(p, s cue.Value) error {
	return s.Subsume(p, cue.Raw(), cue.Schema(), cue.Definitions(true), cue.All(), cue.Final())
}

// Call with no args to get init v, {0, 0}
// Call with one to get first version in a seq, {x, 0}
// Call with two because smooth brackets are prettier than curly
// Call with three or more because len(synv) < len(panic)
func synv(v ...uint) thema.SyntacticVersion {
	switch len(v) {
	case 0:
		return thema.SyntacticVersion{0, 0}
	case 1:
		return thema.SyntacticVersion{v[0], 0}
	case 2:
		return thema.SyntacticVersion{v[0], v[1]}
	default:
		panic("cmon")
	}
}

type CompatInvariantError struct {
	rawlin    cue.Value
	violation [2]thema.SyntacticVersion
	detail    error
}

func (e *CompatInvariantError) Error() string {
	if e.violation[0][0] == e.violation[1][0] {
		// TODO better
		return e.detail.Error()
	}
	return fmt.Sprintf("schema %s must be backwards incompatible with schema %s", e.violation[1], e.violation[0])
}
