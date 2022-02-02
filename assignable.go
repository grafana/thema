package thema

import (
	"bytes"
	"fmt"
	"reflect"

	"cuelang.org/go/cue"
)

// AssignableTo indicates whether all valid instances of the provided Thema
// schema can be assigned to the provided Go type.
//
// If the provided T is a pointer, it will be implicitly dereferenced.
//
// The provided T must necessarily be of struct type, as it is a requirement
// that all Thema schemas are of base type struct.
//
//  type MyType struct {
//  	MyField string `json:"myfield"`
//  }
//
//  AssignableTo(sch, &MyType{})
//
// Assignability rules are specified here: https://github.com/grafana/thema/blob/main/docs/invariants.md#go-assignability
func AssignableTo(sch Schema, T interface{}) error {
	return assignable(sch.UnwrapCUE(), T)
}

const scalarKinds = cue.NullKind | cue.BoolKind |
	cue.IntKind | cue.FloatKind | cue.StringKind | cue.BytesKind

func assignable(sch cue.Value, T interface{}) error {
	pv := reflect.ValueOf(T)

	if pv.Kind() != reflect.Ptr {
		return fmt.Errorf("must provide pointer type, got %T (%s)", T, pv.Kind())
	}

	v := pv.Elem()

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("must provide pointer to struct kind, got *%s", v.Kind())
	}

	ctx := sch.Context()
	gt := ctx.EncodeType(v.Interface())

	// None of the builtin CUE functions do _quite_ what we want here. In the
	// simple case, we might check subsumption of the Go type by the CUE
	// schema, but that falls down because bounds constraints in CUE may be
	// narrower than pure native Go types (e.g. string enums), and Go's type
	// system gives us no choice but to accept that case.
	//
	// We also can't check subsumption of the CUE schema by the Go type, because
	// the addition of `*null |` to every field with a Go pointer type means
	// that many CUE schema values won't be instances of the Go type. (This is
	// something that can be changed in the logic of EncodeType, hopefully.)
	//
	// More importantly, checking subsumption of the schema type by the Go type
	// will not verify that all the schema fields are satisfied/exist - only
	// that the ones that do exist in the Go type are also present and subsumed
	// in the schema type. And that's not even considering optional fields. This
	// makes builtin cue.Value.Subsume() insufficient for our purposes.
	//
	// Forcing the Go type to be closed would plausibly help with all of this,
	// except erroneous nulls. But the above considerations force us to roll our
	// own definition of assignability, at least for now.

	// Errors, keyed by string
	errs := make(assignErrs)

	type checkfn func(gval, sval cue.Value, p cue.Path)
	var check, checkstruct, checklist, checkscalar checkfn

	check = func(gval, sval cue.Value, p cue.Path) {
		// At least for now, we have to deal with these unhelpful *null
		// appearing in the encoding of pointer types.
		gval = stripLeadNull(gval)

		sk, gk := sval.IncompleteKind(), gval.IncompleteKind()
		// strict equality _might_ be too restrictive? But it's better to start there
		if sk != gk {
			errs[p.String()] = fmt.Errorf("%s: is kind %s in schema, but kind %s in Go type", p, sk, gk)
			return
		}

		switch sk {
		case cue.ListKind:
			checklist(gval, sval, p)
		// case cue.NumberKind, cue.FloatKind, cue.IntKind, cue.StringKind, cue.BytesKind, cue.BoolKind:
		case cue.NumberKind, cue.FloatKind, cue.IntKind, cue.StringKind, cue.BytesKind, cue.BoolKind, cue.NullKind:
			checkscalar(gval, sval, p)
		case cue.StructKind:
			checkstruct(gval, sval, p)
		default:
			// if sk & scalarKinds == sk {
			// 	errs[p.String()] = fmt.Errorf("%s: schema is unrepresentable in Go, allows multiple primitive types %s", sk)
			// 	return
			// }
			panic(fmt.Sprintf("unhandled kind %s", sk))
		}
	}

	checkstruct = func(ogval, osval cue.Value, p cue.Path) {
		// Walk the sch side, as we'll allow excess fields on the go side
		ss, gmap := structToSlice(osval), structToMap(ogval)

		// The returned cue.Value appears to differ depending on whether it's
		// accessed through an iterator vs. LookupPath. This matters(?) in the
		// context of doing things like comparing kinds.
		for _, vp := range ss {
			sval, p := vp.Value, cue.MakePath(append(p.Selectors(), vp.Path.Selectors()...)...)
			// Optional() gives us paths that are either optional or not, which
			// seems reasonable at least until we really formally define this
			// relation
			gval, exists := gmap[vp.Path.Optional().String()]

			// TODO replace these one-offs with formalized error types
			if !exists {
				errs[p.String()] = fmt.Errorf("%s: absent from Go type", p)
				continue
			}
			check(gval, sval, p)
		}

		// TODO check for additional fields on Go side
	}

	checklist = func(gval, sval cue.Value, p cue.Path) {
		var los, log cue.Value
		glen, slen := gval.Len(), sval.Len()
		// Ensure alignment of list openness/closedness
		if glen.IsConcrete() != slen.IsConcrete() {
			if slen.IsConcrete() {
				errs[p.String()] = fmt.Errorf("%s: list is closed in schema, Go type must be an array, not slice", p)
			} else {
				errs[p.String()] = fmt.Errorf("%s: list is open in schema, Go type must be a slice, not array", p)
			}
			return
		}

		if err := glen.Subsume(slen); err != nil {
			// should be unreachable?
			errs[p.String()] = fmt.Errorf("%s: incompatible list lengths in schema (%s) and Go type (%s)", p, slen, glen)
			return
		}

		if glen.IsConcrete() {
			if ilen, err := slen.Int64(); err != nil {
				panic(fmt.Errorf("unreachable: %w", err))
			} else if ilen == 0 {
				// empty list on both sides - weird, but not illegal
				return
			}
			// Go's type system guarantees that all list elements will be of the
			// same type, so as long as all the CUE list elements are the same,
			// then comparing should be safe.  Of course, checking "sameness" of
			// incomplete values isn't (?) trivial. Mutual subsume...
			iter, err := sval.List()
			if err != nil {
				panic(err)
			}

			iter.Next()
			lastsel, lastval := iter.Selector(), iter.Value()

			for iter.Next() {
				los = iter.Value() // it's fine to just keep updating the reference
				// Failures indicate the CUE schema is unrepresentable in Go.
				// That's the kind of thing we'd likely prefer to know/have in
				// some more universal place.
				lerr, rerr := lastval.Subsume(los, cue.Schema()), los.Subsume(lastval, cue.Schema())
				if lerr != nil || rerr != nil {
					fmt.Println(lerr, rerr)
					errs[p.String()] = fmt.Errorf("%s: schema is list of multiple types; not representable in Go", p)
					return
				}

				lastsel, lastval = iter.Selector(), iter.Value()
			}
			p = cue.MakePath(append(p.Selectors(), lastsel)...)

			iter, err = gval.List()
			if err != nil {
				panic(err)
			}
			_, log = iter.Next(), iter.Value()
		} else {
			los = sval.LookupPath(cue.MakePath(cue.AnyIndex))
			log = gval.LookupPath(cue.MakePath(cue.AnyIndex))
			p = cue.MakePath(append(p.Selectors(), cue.AnyIndex)...)
		}
		check(log, los, p)
	}

	checkscalar = func(gval, sval cue.Value, p cue.Path) {
		// Because the CUE types can have narrower bounds, and we're
		// really interested in whether all valid schema instances will
		// be assignable to the Go type, we have to see if the Go type
		// subsumes the schema, rather than the more intuitive check
		// that the schema subsumes the Go type.
		if err := gval.Subsume(sval, cue.Schema()); err != nil {
			errs[p.String()] = fmt.Errorf("%s: %v not an instance of %v", p, sval, gval)
		}
	}

	// Walk down the whole struct tree
	check(gt, sch, cue.MakePath())

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func stripLeadNull(v cue.Value) cue.Value {
	if op, vals := v.Expr(); op == cue.OrOp && vals[0].Null() == nil {
		return vals[1]
	}
	return v
}

type valpath struct {
	Path  cue.Path
	Value cue.Value
}

type structSlice []valpath

func structToMap(v cue.Value) map[string]cue.Value {
	m := make(map[string]cue.Value)
	iter, err := v.Fields(cue.Optional(true))
	if err != nil {
		panic(err)
	}

	for iter.Next() {
		// fmt.Printf("sm %v %#v\n", iter.Selector(), iter.Value())
		m[iter.Selector().String()] = iter.Value()
		m[iter.Selector().Optional().String()] = iter.Value()
	}

	return m
}

func structToSlice(v cue.Value) structSlice {
	var ss structSlice
	iter, err := v.Fields(cue.Optional(true))
	if err != nil {
		panic(err)
	}

	for iter.Next() {
		// fmt.Printf("ss %v %#v\n", iter.Selector(), iter.Value())
		ss = append(ss, valpath{
			Path:  cue.MakePath(iter.Selector()),
			Value: iter.Value(),
		})
	}

	return ss
}

type assignErrs map[string]error

func (m assignErrs) Error() string {
	var buf bytes.Buffer

	var i int
	for _, err := range m {
		if i == len(m)-1 {
			fmt.Fprint(&buf, err)
		} else {
			fmt.Fprint(&buf, err, "\n")
		}
		i++
	}

	return (&buf).String()
}
