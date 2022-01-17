package kernel

import (
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
	// Try to dodge around the *null we get back by pulling out the latter part of the expr
	op, vals := tv.Expr()
	if op != cue.OrOp {
		panic("not an or")
	}
	realval := vals[1]
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
