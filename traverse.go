package thema

import (
	"errors"
	"fmt"
	"math/bits"

	"cuelang.org/go/cue"
)

// SearchAndValidate traverses the family of schemas reachable from the provided
// Schema. For each schema, it attempts to validate the provided
// value, which may be a byte slice representing valid JSON (TODO YAML), a Go
// struct, or cue.Value. If providing a cue.Value that is not fully concrete,
// the result is undefined.
//
// Traversal is performed from the newest schema to the oldest. However, because
// newer Schema have no way of directly accessing their predecessors
// (they form a singly-linked list), the oldest possible schema should always be
// provided - typically, the one returned from the family loader function.
//
// Failure to validate against any schema in the family is indicated by a
// non-nil error return. Success is indicated by a non-nil Schema.
// If successful, the returned Schema will be the first one against
// which the provided resource passed validation.
func SearchAndValidate(s Schema, v cue.Value) (Schema, error) {
	arr := AsArray(s)

	// Work from latest to earliest
	var err error
	for o := len(arr) - 1; o >= 0; o-- {
		for i := len(arr[o]) - 1; i >= 0; i-- {
			if _, err = arr[o][i].Validate(v); err == nil {
				return arr[o][i], nil
			}
		}
	}

	// TODO sloppy, return more than last error. Need our own error type that
	// collates all the individual errors, relates them to the schema that
	// produced them, and ideally deduplicates repeated errors across each
	// schema.
	return nil, err
}

// AsArray collates all Schema in a lineage into a two-dimensional
// array. The outer array index corresponds to sequence number and inner
// array index to schema version number.
func AsArray(sch Schema) [][]Schema {
	var ret [][]Schema
	var flat []Schema

	// two loops. lazy day, today
	for sch != nil {
		flat = append(flat, sch)
		sch = sch.Successor()
	}

	for _, sch := range flat {
		seqv := int(sch.Version()[0])
		if len(ret) == seqv {
			ret = append(ret, []Schema{})
		}
		ret[seqv] = append(ret[seqv], sch)
	}

	return ret
}

// Find traverses the chain of Schema until the criteria in the
// SearchOption is met.
//
// If no schema is found that fulfills the criteria, nil is returned. Latest()
// and LatestInCurrentMajor() will always succeed, unless the input schema is
// nil.
func Find(s Schema, opt SearchOption) Schema {
	if s == nil {
		return nil
	}

	p := &ssopt{}
	opt(p)
	if err := p.validate(); err != nil {
		panic(fmt.Sprint("unreachable:", err))
	}

	switch {
	case p.latest:
		for ; s.Successor() != nil; s = s.Successor() {
		}
		return s

	case p.latestInCurrentMajor:
		p.latestInMajor = s.Version()[0]
		fallthrough

	case p.hasLatestInMajor:
		iseqv := s.Version()[0]
		if iseqv > p.latestInMajor {
			return nil
		}

		var last Schema
		for iseqv <= p.latestInMajor {
			last, s = s, s.Successor()
			if s == nil {
				if iseqv == p.latestInMajor {
					return last
				}
				return nil
			}

			iseqv = s.Version()[0]
		}
		return last

	default: // exact
		for s != nil {
			if p.exact == s.Version() {
				return s
			}
			s = s.Successor()
		}
		return nil
	}
}

// SearchOption indicates how far along a chain of schemas an operation should
// proceed.
type SearchOption sso

type sso func(p *ssopt)

type ssopt struct {
	latest               bool
	latestInMajor        uint
	hasLatestInMajor     bool
	latestInCurrentMajor bool
	exact                SyntacticVersion
}

func (p *ssopt) validate() error {
	var which uint16
	if p.latest {
		which = which + 1<<1
	}
	if p.exact != [2]uint{0, 0} {
		which = which + 1<<2
	}
	if p.hasLatestInMajor {
		which = which + 1<<3
	} else if p.latestInMajor != 0 {
		// Disambiguate real zero from default zero
		return fmt.Errorf("latestInMajor should never be non-zero if hasLatestInMajor is false, got %v", p.latestInMajor)
	}
	if p.latestInCurrentMajor {
		which = which + 1<<4
	}

	if bits.OnesCount16(which) != 1 {
		return errors.New("may only pass one SchemaSearchOption")
	}
	return nil
}

// Latest indicates that traversal will continue to the newest schema in the
// newest sequence.
func Latest() SearchOption {
	return func(p *ssopt) {
		p.latest = true
	}
}

// LatestInMajor will find the latest schema within the provided major version
// sequence. If no sequence exists corresponding to the provided number, traversal
// will terminate with an error.
func LatestInMajor(seqv uint) SearchOption {
	return func(p *ssopt) {
		p.latestInMajor = seqv
	}
}

// LatestInCurrentMajor will find the newest schema having the same major
// version as the schema from which the search begins.
func LatestInCurrentMajor() SearchOption {
	return func(p *ssopt) {
		p.latestInCurrentMajor = true
	}
}

// Exact will find the schema with the exact major and minor version number
// provided.
func Exact(v SyntacticVersion) SearchOption {
	return func(p *ssopt) {
		p.exact = v
	}
}
