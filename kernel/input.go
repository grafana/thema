package kernel

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/encoding/gocode/gocodec"
	"cuelang.org/go/encoding/json"
	"github.com/grafana/thema"
)

// A TypeFactory must emit a pointer to the Go type that a kernel will
// ultimately produce as output.
//
// TODO the function accomplished by this should be trivial to achieve with generics...?
type TypeFactory func() interface{}

// A Decoder takes some input data as an []byte and loads it into a
// cue.Value.
type Decoder func(*cue.Context, []byte) (cue.Value, error)

// NewJSONByteDecoder creates a Decoder func that translates some JSON data
// input into a cue.Value.
//
// The provided path is used as the sourcename for the input data (the
// identifier for the data used by CUE error messages). Any provided
// cue.BuildOptions are passed along to cue.Context.BuildExpr().
func NewJSONByteDecoder(path string, o ...cue.BuildOption) Decoder {
	return func(ctx *cue.Context, data []byte) (cue.Value, error) {
		expr, err := json.Extract(path, data)
		if err != nil {
			return cue.Value{}, err
		}
		return ctx.BuildExpr(expr, o...), nil
	}
}

// InputKernelConfig holds configuration options for InputKernel.
type InputKernelConfig struct {
	// TypeFactory determines the Go type that processed values will be
	// loaded into by Converge(). It must return a pointer to a Go type that is
	// valid with respect to the `to` schema in the `lin` lineage.
	//
	// Attempting to create an InputKernel with a nil TypeFactory will panic.
	TypeFactory TypeFactory

	// Decoder accepts input data and converts it to a cue.Value.
	//
	// Attempting to create an InputKernel with a nil Decoder will panic.
	Decoder Decoder

	// Lineage is the Thema lineage containing the schema for data to be validated
	// against and translated through.
	//
	// Attempting to create an InputKernel with a nil Lineage will panic.
	Lineage thema.Lineage

	// To is the schema version on which all operations will converge.
	To thema.SyntacticVersion

	// TODO add something for interrupting/mediating translation vis-a-vis accumulated lacunae
}

// An InputKernel accepts all the valid inputs for a given lineage, converges
// them onto a single statically-chosen schema version via Thema translation,
// and emits the result in a native Go type.
type InputKernel struct {
	init   bool
	tf     TypeFactory
	decode Decoder
	lin    thema.Lineage
	to     thema.SyntacticVersion
	// TODO add something for interrupting/mediating translation vis-a-vis accumulated lacunae
}

// NewInputKernel constructs an input kernel.
//
// InputKernels accepts input data in whatever format (e.g. JSON, YAML)
// supported by its Decoder, validates the data, translates it to a single target version, then
func NewInputKernel(cfg InputKernelConfig) (InputKernel, error) {
	if cfg.Lineage == nil {
		panic("must provide a non-nil Lineage")
	}
	if cfg.TypeFactory == nil {
		panic("must provide a non-nil TypeFactory")
	}
	if cfg.Decoder == nil {
		panic("must provide a non-nil Decoder")
	}

	// The concurrency warnings in the docs on Codec are concerning - don't use
	// the Runtime concurrently for any other operations, but concurrent use of
	// only the codec is fine? ugh, that wouldn't be easy to coordinate under the
	// best of circumstances. And there's really nothing we can do in a
	// situation like this. So...guess we're YOLOing it for now?
	codec := gocodec.New((*cue.Runtime)(cfg.Lineage.UnwrapCUE().Context()), nil)
	sch, err := cfg.Lineage.Schema(cfg.To)
	if err != nil {
		return InputKernel{}, err
	}

	// Verify that the input Go type is valid with respect to the indicated
	// schema. Effect is that the caller cannot get an InputKernel without a
	// valid Go type to write to.
	//
	// TODO verify this is actually how we check this
	if err = codec.Validate(sch.UnwrapCUE(), cfg.TypeFactory()); err != nil {
		return InputKernel{}, err
	}

	return InputKernel{
		init:   true,
		tf:     cfg.TypeFactory,
		decode: cfg.Decoder,
		lin:    cfg.Lineage,
		to:     cfg.To,
	}, nil
}

// Converge runs input data through the full kernel process: validate, translate to a
// fixed version, return transformed instance along with any emitted lacunae.
//
// Valid formats for the input data are determined by the Decoder func with which
// the kernel was constructed. Invalid data will result in an error.
//
// Type safety of the return value is guaranteed by checks performed in
// NewInputKernel() - if error is non-nil, the first return value is guaranteed
// to be an instance of the type returned from the TypeFactory with which the
// kernel was constructed.
//
// It is safe to call Converge frm multiple goroutines.
func (k InputKernel) Converge(data []byte) (interface{}, thema.TranslationLacunae, error) {
	if !k.init {
		panic("kernel not initialized")
	}

	// Decode the input data into a cue.Value
	v, err := k.decode(k.lin.UnwrapCUE().Context(), data)
	if err != nil {
		return nil, nil, err
	}

	// Validate that the data constitutes an instance of at least one of the schemas in the lineage
	inst := k.lin.ValidateAny(v)
	if err != nil {
		return nil, nil, err
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

// Config returns a copy of the kernel's active configuration.
func (k InputKernel) Config() InputKernelConfig {
	if !k.init {
		panic("kernel not initialized")
	}

	return InputKernelConfig{
		TypeFactory: k.tf,
		Decoder:     k.decode,
		Lineage:     k.lin,
		To:          k.to,
	}
}
