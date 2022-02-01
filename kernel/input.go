package kernel

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"

	"cuelang.org/go/cue"
	"github.com/grafana/thema"
)

// InputKernelConfig holds configuration options for InputKernel.
type InputKernelConfig struct {
	// TypeFactory determines the Go type that processed values will be loaded
	// into by Converge(). It must return a non-pointer Go value that is valid
	// with respect to the `to` schema in the `lin` lineage.
	//
	// Attempting to create an InputKernel with a nil TypeFactory will panic.
	//
	// TODO investigate if we can allow pointer types without things getting weird
	TypeFactory TypeFactory

	// Loader takes input data and converts it to a cue.Value.
	//
	// Attempting to create an InputKernel with a nil Loader will panic.
	Loader DataLoader

	// Lineage is the Thema lineage containing the schema for data to be validated
	// against and translated through.
	//
	// Attempting to create an InputKernel with a nil Lineage will panic.
	Lineage thema.Lineage

	// To is the schema version on which all operations will converge.
	To thema.SyntacticVersion

	// TODO add something for interrupting/mediating translation vis-a-vis accumulated lacunas
}

// An InputKernel accepts all the valid inputs for a given lineage, converges
// them onto a single statically-chosen schema version via Thema translation,
// and emits the result in a native Go type.
type InputKernel struct {
	init bool
	tf   TypeFactory
	// whether or not the TypeFactory returns a pointer type
	ptrtype bool
	load    DataLoader
	lin     thema.Lineage
	to      thema.SyntacticVersion
	// TODO add something for interrupting/mediating translation vis-a-vis accumulated lacunas
}

// NewInputKernel constructs an input kernel.
//
// InputKernels accepts input data in whatever format (e.g. JSON, YAML)
// supported by its DataLoader, validates the data, translates it to a single target version, then
func NewInputKernel(cfg InputKernelConfig) (InputKernel, error) {
	if cfg.Lineage == nil {
		panic("must provide a non-nil Lineage")
	}
	if cfg.TypeFactory == nil {
		panic("must provide a non-nil TypeFactory")
	}
	if cfg.Loader == nil {
		panic("must provide a non-nil Decoder")
	}

	sch, err := cfg.Lineage.Schema(cfg.To)
	if err != nil {
		return InputKernel{}, err
	}

	t := cfg.TypeFactory()
	// Ensure that the type returned from the TypeFactory is a pointer. If it's
	// not, we can't create a pointer to it in a way that's necessary for
	// decoding later. Downside is that having it be a pointer means it allows a
	// null, which isn't what we want.
	if k := reflect.ValueOf(t).Kind(); k != reflect.Ptr {
		return InputKernel{}, fmt.Errorf("cfg.TypeFactory must return a pointer type, got %T (%s)", t, k)
	}

	// Verify that the input Go type is valid with respect to the indicated
	// schema. Effect is that the caller cannot get an InputKernel without a
	// valid Go type to write to.
	tv := cfg.Lineage.UnwrapCUE().Context().EncodeType(t)
	// fmt.Println(tv)
	// Try to dodge around the *null we get back by pulling out the latter part of the expr
	op, vals := tv.Expr()
	if op != cue.OrOp {
		panic("not an or")
	}
	realval := vals[1]
	// fmt.Println("stripped", realval)
	if err := sch.UnwrapCUE().Subsume(realval, cue.Schema(), cue.Raw()); err != nil {
		return InputKernel{}, err
	}

	return InputKernel{
		init: true,
		tf:   cfg.TypeFactory,
		load: cfg.Loader,
		lin:  cfg.Lineage,
		to:   cfg.To,
	}, nil
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
	gval := ctx.EncodeType(v.Interface())

	// None of the builtin functions do _quite_ what we want here. In the simple
	// case, we'd want to check subsumption of the Go type by the CUE schema,
	// but that falls down because bounds constraints in CUE may be narrower
	// than pure native Go types (e.g. string enums), and Go's type system gives
	// us no choice but to accept that case.
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
	// makes it flatly insufficient for our purposes.
	//
	// Forcing the Go type to be closed would plausibly help with all of this,
	// except erroneous nulls. But the above considerations force us to roll our
	// own definition of assignability, at least for now.

	// First, try unification. This covers the big, egregious cases (e.g. int
	// vs. string for a given field), though it gives us a mash of errors.
	// if err := sch.Unify(gval).Validate(cue.All()); err != nil {
	// 	return err
	// }

	// Errors, keyed by string
	errs := make(assignErrs)

	type walkfn func(gval, sval cue.Value, sel ...cue.Selector)
	var walk walkfn

	walk = func(ogval, osval cue.Value, sel ...cue.Selector) {
		// Walk the sch side, as we'll allow excess fields on the go side
		ss, gmap := structToSlice(osval), structToMap(ogval)

		// The returned cue.Value appears to differ depending on whether it's
		// accessed through an iterator vs. LookupPath. This matters(?) in the
		// context of doing things like comparing kinds.

		for _, vp := range ss {
			sval, p := vp.Value, cue.MakePath(append(sel, vp.Path.Selectors()...)...)
			// Optional() gives us paths that are either optional or not, which
			// seems reasonable at least until we really formally define this
			// relation
			gval, exists := gmap[vp.Path.Optional().String()]

			// TODO replace these one-offs with formalized error types
			if !exists {
				errs[p.String()] = fmt.Errorf("%s: absent from Go type", p)
				continue
			}
			// At least for now, we have to deal with these unhelpful *null
			// appearing in the encoding of pointer types.
			gval = stripLeadNull(gval)

			sk, gk := sval.IncompleteKind(), gval.IncompleteKind()
			// strict equality _might_ be too restrictive? But it's better to start there
			if sk != gk {
				errs[p.String()] = fmt.Errorf("%s: of kind %s in schema, but kind %s in Go type", p, sk, gk)
				continue
			}

			switch sk {
			case cue.ListKind:
				glen, slen := gval.Len(), sval.Len()
				// Ensure alignment of list openness/closedness
				if glen.IsConcrete() != slen.IsConcrete() {
					if slen.IsConcrete() {
						errs[p.String()] = fmt.Errorf("%s: list is closed in schema, Go type must be an array, not slice", p)
					} else {
						errs[p.String()] = fmt.Errorf("%s: list is open in schema, Go type must be a slice, not array", p)
					}
				}

				if err := glen.Subsume(slen); err != nil {
					// should be unreachable?
					errs[p.String()] = fmt.Errorf("%s: incompatible list lengths in schema (%s) and Go type (%s)", p, slen, glen)
					continue
				}

				if glen.IsConcrete() {
					if ilen, err := slen.Int64(); err != nil {
						panic(fmt.Errorf("unreachable: %w", err))
					} else if ilen == 0 {
						// empty list on both sides - weird, but not illegal
						continue
					}
					// Go's type system guarantees that all list elements will
					// be of the same type, so as long as all the CUE list
					// elements are the same, then comparing should be safe.  Of
					// course, checking "sameness" of incomplete values isn't
					// (?) trivial. Mutual subsume...
					iter, err := sval.List()
					if err != nil {
						panic(err)
					}

					iter.Next()
					lastsel, lastval := iter.Selector(), iter.Value()
					sval = iter.Value()

					for iter.Next() {
						sval = iter.Value() // it's fine to just keep updating the reference
						// Failures indicate the CUE schema is unrepresentable
						// in Go. That's the kind of thing we'd likely prefer to
						// know/have in some more universal place.
						lerr, rerr := lastval.Subsume(sval, cue.Schema()), sval.Subsume(lastval, cue.Schema())
						if lerr != nil || rerr != nil {
							errs[p.String()] = fmt.Errorf("%s: schema is list of multiple types; not representable in Go", p)
							continue
						}

						lastsel, lastval = iter.Selector(), iter.Value()
					}
					p = cue.MakePath(append(p.Selectors(), lastsel)...)

					iter, err = gval.List()
					if err != nil {
						panic(err)
					}
					_, gval = iter.Next(), iter.Value()
				} else {
					sval = sval.LookupPath(cue.MakePath(cue.AnyIndex))
					gval = gval.LookupPath(cue.MakePath(cue.AnyIndex))
					p = cue.MakePath(append(p.Selectors(), cue.AnyIndex)...)
				}

				if jk := sval.IncompleteKind() & gval.IncompleteKind(); jk == 0 {
					errs[p.String()] = fmt.Errorf("%s: of kind %s in schema, but kind %s in Go type", p, sval.IncompleteKind(), gval.IncompleteKind())
					continue
				} else if jk&scalarKinds == 0 {
					walk(gval, sval, p.Selectors()...)
					continue
				}

				// They're both scalars of the same incomplete kind, handle them
				// without descending
				fallthrough
			case cue.NumberKind, cue.FloatKind, cue.IntKind, cue.StringKind, cue.BytesKind, cue.BoolKind:
				// Because the CUE types can have narrower bounds, and we're
				// really interested in whether all valid schema instances will
				// be assignable to the Go type, we have to see if the Go type
				// subsumes the schema, rather than the more intuitive check
				// that the schema subsumes the Go type.
				if err := gval.Subsume(sval, cue.Schema()); err != nil {
					errs[p.String()] = fmt.Errorf("%s: %v not an instance of %v", p, sval, gval)
				}
			case cue.StructKind:
				walk(gval, sval, p.Selectors()...)
			default:
				panic(fmt.Sprintf("unhandled kind %s", sk))
			}
		}
	}

	walk(gval, sch)

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

// Converge runs input data through the full kernel process: validate, translate to a
// fixed version, return transformed instance along with any emitted lacunas.
//
// Valid formats for the input data are determined by the DataLoader func with which
// the kernel was constructed. Invalid data will result in an error.
//
// Type safety of the return value is guaranteed by checks performed in
// NewInputKernel(). If error is non-nil, the concrete type of the first return
// value is guaranteed to be the type returned from the TypeFactory with
// which the kernel was constructed.
//
// It is safe to call Converge from multiple goroutines.
func (k InputKernel) Converge(data []byte) (interface{}, thema.TranslationLacunas, error) {
	if !k.init {
		panic("kernel not initialized")
	}

	// Decode the input data into a cue.Value
	v, err := k.load(k.lin.UnwrapCUE().Context(), data)
	if err != nil {
		// TODO wrap error for use with errors.Is
		return nil, nil, err
	}

	// Validate that the data constitutes an instance of at least one of the schemas in the lineage
	inst := k.lin.ValidateAny(v)
	if inst == nil {
		// TODO wrap error for use with errors.Is
		return nil, nil, errors.New("validation failed")
	}

	transval, lac := inst.Translate(k.to)
	ret := k.tf()
	transval.UnwrapCUE().Decode(ret)
	return ret, lac, nil
}

// IsInitialized reports whether the InputKernel has been properly initialized.
//
// Calling methods on an uninitialized kernel results in a panic. Kernels can
// only be initialized through NewInputKernel.
func (k InputKernel) IsInitialized() bool {
	return k.init
}

// Config returns a copy of the kernel's configuration.
func (k InputKernel) Config() InputKernelConfig {
	if !k.init {
		panic("kernel not initialized")
	}

	return InputKernelConfig{
		TypeFactory: k.tf,
		Loader:      k.load,
		Lineage:     k.lin,
		To:          k.to,
	}
}

// type ErrDataUnloadable struct {

// }

// type ErrInvalidData struct {

// }
