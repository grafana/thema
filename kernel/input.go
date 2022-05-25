package kernel

import (
	"fmt"
	"reflect"

	"github.com/grafana/thema"
)

// InputKernelConfig holds configuration options for InputKernel.
type InputKernelConfig struct {
	// Typ is the Go type that processed values will be loaded
	// into by Converge(). It must return a non-pointer Go value that is valid
	// with respect to the `to` schema in the `lin` lineage.
	//
	// Attempting to create an InputKernel with a nil Typ will panic.
	//
	// TODO investigate if we can allow pointer types without things getting weird
	Typ interface{}

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
	typ  interface{}
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
	if cfg.Typ == nil {
		panic("must provide a non-nil TypeFactory")
	}
	if cfg.Loader == nil {
		panic("must provide a non-nil Decoder")
	}

	sch, err := cfg.Lineage.Schema(cfg.To)
	if err != nil {
		return InputKernel{}, err
	}

	// Ensure that the type returned from the TypeFactory is a pointer. If it's
	// not, we can't create a pointer to it in a way that's necessary for
	// decoding later. Downside is that having it be a pointer means it allows a
	// null, which isn't what we want.
	if k := reflect.ValueOf(cfg.Typ).Kind(); k != reflect.Ptr {
		return InputKernel{}, fmt.Errorf("cfg.TypeFactory must return a pointer type, got %T (%s)", cfg.Typ, k)
	}

	// Verify that the input Go type is valid with respect to the indicated
	// schema. Effect is that the caller cannot get an InputKernel without a
	// valid Go type to write to.
	if err := thema.AssignableTo(sch, cfg.Typ); err != nil {
		return InputKernel{}, err
	}

	return InputKernel{
		init: true,
		typ:  cfg.Typ,
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

	transval, lac, err := k.process(data)
	if err != nil {
		return nil, nil, err
	}

	ret := k.typ
	transval.UnwrapCUE().Decode(ret)
	return ret, lac, nil
}

// ConvergeJSON is the same as Converge, but emits the translated instance as
// byte slice of JSON rather than unmarshalling to a Go type.
func (k InputKernel) ConvergeJSON(data []byte) ([]byte, thema.TranslationLacunas, error) {
	if !k.init {
		panic("kernel not initialized")
	}

	transval, lac, err := k.process(data)
	if err != nil {
		return nil, nil, err
	}

	b, err := transval.UnwrapCUE().MarshalJSON()
	return b, lac, err
}

func (k InputKernel) process(data []byte) (*thema.Instance, thema.TranslationLacunas, error) {
	// Decode the input data into a cue.Value
	v, err := k.load(k.lin.UnwrapCUE().Context(), data)
	if err != nil {
		// TODO wrap error for use with errors.Is
		return nil, nil, err
	}

	inst := k.lin.ValidateAny(v)
	if inst == nil {
		targetSchema, err := k.lin.Schema(k.to)
		if err != nil {
			return nil, nil, err
		}
		if _, err := targetSchema.Validate(v); err != nil {
			return nil, nil, err
		}
		return nil, nil, fmt.Errorf("validation failed")
	}

	tinst, lac := inst.Translate(k.to)
	return tinst, lac, nil
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
		Typ:     k.typ,
		Loader:  k.load,
		Lineage: k.lin,
		To:      k.to,
	}
}

// type ErrDataUnloadable struct {

// }

// type ErrInvalidData struct {

// }
