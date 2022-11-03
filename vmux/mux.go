package vmux

import (
	"encoding/json"
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	cjson "cuelang.org/go/encoding/json"
	"cuelang.org/go/encoding/yaml"
	pyaml "cuelang.org/go/pkg/encoding/yaml"
	"github.com/grafana/thema"
)

// UntypedMux is a version multiplexer that maps a []byte containing data at any
// schematized version to a [thema.Instance] at a particular schematized version.
type UntypedMux func(b []byte) (*thema.Instance, thema.TranslationLacunas, error)

// NewUntypedMux creates an [UntypedMux] from the provided [thema.Schema].
//
// When the returned mux func is called, it will:
//
//   - Decode the input []byte using the provided [Decoder], then
//   - Pass the result to [thema.Schema.Validate], then
//   - Call [thema.Instance.Translate] on the result, to the version of the provided [thema.Schema], then
//   - Return the resulting [thema.Instance], [thema.TranslationLacunas], and error
//
// The returned error may be from any of the above steps.
func NewUntypedMux(sch thema.Schema, dec Decoder) UntypedMux {
	ctx := sch.Lineage().Underlying().Context()
	// Prepare no-match error string once for reuse
	vstring := allvstr(sch)

	return func(b []byte) (*thema.Instance, thema.TranslationLacunas, error) {
		v, err := dec.Decode(ctx, b)
		if err != nil {
			// TODO wrap error for use with errors.Is
			return nil, nil, err
		}

		// Try the given schema first, on the premise that in general it's the
		// most likely one for an application to encounter
		tinst, err := sch.Validate(v)
		if err == nil {
			return tinst, nil, nil
		}

		// Walk in reverse order on the premise that, in general, newer versions are more
		// likely to be provided than older versions
		isch := latest(sch.Lineage())
		for ; isch != nil; isch = isch.Predecessor() {
			if isch.Version() == sch.Version() {
				continue
			}

			if inst, ierr := isch.Validate(v); ierr == nil {
				trinst, lac := inst.Translate(sch.Version())
				return trinst, lac, nil
			}
		}

		return nil, nil, fmt.Errorf("data invalid against all versions (%s), error against %s: %w", vstring, sch.Version(), err)
	}
}

// ByteMux is a version multiplexer that maps a []byte containing data at any
// schematized version to a []byte containing data at a particular schematized version.
type ByteMux func(b []byte) ([]byte, thema.TranslationLacunas, error)

// NewByteMux creates a [ByteMux] func from the provided [thema.Schema].
//
// When the returned mux func is called, it will:
//
//   - Decode the input []byte using the provided [Endec], then
//   - Pass the result to [thema.Schema.Validate], then
//   - Call [thema.Instance.Translate] on the result, to the version of the provided [thema.Schema], then
//   - Encode the resulting [thema.Instance] to a []byte, then
//   - Return the resulting []byte, [thema.TranslationLacunas], and error
//
// The returned error may be from any of the above steps.
func NewByteMux(sch thema.Schema, end Endec) ByteMux {
	f := NewUntypedMux(sch, end)
	return func(b []byte) ([]byte, thema.TranslationLacunas, error) {
		ti, lac, err := f(b)
		if err != nil {
			return nil, lac, err
		}
		ob, err := end.Encode(ti.Underlying())
		return ob, lac, err
	}
}

// ValueMux is a version multiplexer that maps a []byte containing data at any
// schematized version to a Go var of a type that a particular schematized
// version is [thema.AssignableTo].
type ValueMux[T thema.Assignee] func(b []byte) (T, thema.TranslationLacunas, error)

// NewValueMux creates a [ValueMux] func from the provided [thema.TypedSchema].
//
// When the returned mux func is called, it will:
//
//   - Decode the input []byte using the provided [Decoder], then
//   - Pass the result to [thema.TypedSchema.ValidateTyped], then
//   - Call [thema.Instance.Translate] on the result, to the version of the provided [thema.TypedSchema], then
//   - Populate an instance of T by calling [thema.TypedInstance.Value] on the result, then
//   - Return the resulting T, [thema.TranslationLacunas], and error
//
// The returned error may be from any of the above steps.
func NewValueMux[T thema.Assignee](sch thema.TypedSchema[T], dec Decoder) ValueMux[T] {
	f := NewTypedMux[T](sch, dec)
	return func(b []byte) (T, thema.TranslationLacunas, error) {
		ti, lac, err := f(b)
		if err != nil {
			return sch.NewT(), lac, err
		}
		t, err := ti.Value()
		return t, lac, err
	}
}

// TypedMux is a version multiplexer that maps a []byte containing data at any
// schematized version to a [thema.TypedInstance] at a particular schematized version.
type TypedMux[T thema.Assignee] func(b []byte) (*thema.TypedInstance[T], thema.TranslationLacunas, error)

// NewTypedMux creates a [TypedMux] func from the provided [thema.TypedSchema].
//
// When the returned mux func is called, it will:
//
//   - Decode the input []byte using the provided [Decoder], then
//   - Pass the result to [thema.TypedSchema.ValidateTyped], then
//   - Call [thema.Instance.Translate] on the result, to the version of the provided [thema.TypedSchema], then
//   - Return the resulting [thema.TypedInstance], [thema.TranslationLacunas], and error
//
// The returned error may be from any of the above steps.
func NewTypedMux[T thema.Assignee](sch thema.TypedSchema[T], dec Decoder) TypedMux[T] {
	ctx := sch.Lineage().Underlying().Context()
	// Prepare no-match error string once for reuse
	vstring := allvstr(sch)

	return func(b []byte) (*thema.TypedInstance[T], thema.TranslationLacunas, error) {
		v, err := dec.Decode(ctx, b)
		if err != nil {
			// TODO wrap error for use with errors.Is
			return nil, nil, err
		}

		// Try the given schema first, on the premise that in general it's the
		// most likely one for an application to encounter
		tinst, err := sch.ValidateTyped(v)
		if err == nil {
			return tinst, nil, nil
		}

		// Walk in reverse order on the premise that, in general, newer versions are more
		// likely to be provided than older versions
		isch := latest(sch.Lineage())
		for ; isch != nil; isch = isch.Predecessor() {
			if isch.Version() == sch.Version() {
				continue
			}

			if inst, ierr := isch.Validate(v); ierr == nil {
				trinst, lac := inst.Translate(sch.Version())
				tinst, err := thema.BindInstanceType(trinst, sch)
				if err != nil {
					panic(fmt.Errorf("unreachable, instance type should always be bindable: %w", err))
				}
				return tinst, lac, nil
			}
		}

		return nil, nil, fmt.Errorf("data invalid against all versions (%s), error against %s: %w", vstring, sch.Version(), err)
	}
}

func allvstr(sch thema.Schema) string {
	var vl []string
	for isch := thema.SchemaP(sch.Lineage(), thema.SV(0, 0)); isch != nil; isch = isch.Successor() {
		vl = append(vl, isch.Version().String())
	}
	return strings.Join(vl, ", ")
}

func latest(lin thema.Lineage) thema.Schema {
	return thema.SchemaP(lin, thema.LatestVersion(lin))
}

// A Decoder can decode a []byte in a particular format (e.g. JSON, YAML) into a
// [cue.Value], readying it for a call to [thema.Schema.Validate].
type Decoder interface {
	Decode(ctx *cue.Context, b []byte) (cue.Value, error)
}

// An Encoder can encode a [thema.Instance] to a []byte in a particular format
// (e.g. JSON, YAML).
type Encoder interface {
	Encode(cue.Value) ([]byte, error)
}

// An Endec (encoder + decoder) can decode a []byte in a particular format (e.g.
// JSON, YAML) into CUE, and decode from a [thema.Instance] back into a []byte.
//
// It is customary, but not necessary, that an Endec's input and output formats
// are the same.
type Endec interface {
	Decoder
	Encoder
}

type jsonEndec struct {
	path string
}

// NewJSONEndec creates an [Endec] that decodes from and encodes to a JSON []byte.
//
// The provided path is used as the CUE source path for each []byte input
// passed through the decoder. These paths do not affect behavior, but show up
// in error output (e.g. validation).
func NewJSONEndec(path string) Endec {
	return jsonEndec{
		path: path,
	}
}

func (e jsonEndec) Decode(ctx *cue.Context, data []byte) (cue.Value, error) {
	expr, err := cjson.Extract(e.path, data)
	if err != nil {
		return cue.Value{}, err
	}
	return ctx.BuildExpr(expr), nil
}

func (e jsonEndec) Encode(v cue.Value) ([]byte, error) {
	return json.Marshal(v)
}

type yamlEndec struct {
	path string
}

// NewYAMLEndec creates an Endec that decodes from and encodes to a YAML []byte.
//
// The provided path is used as the CUE source path for each []byte input
// passed through the decoder. These paths do not affect behavior, but show up
// in error output (e.g. validation).
func NewYAMLEndec(path string) Endec {
	return yamlEndec{
		path: path,
	}
}

func (e yamlEndec) Decode(ctx *cue.Context, data []byte) (cue.Value, error) {
	expr, err := yaml.Extract(e.path, data)
	if err != nil {
		return cue.Value{}, err
	}
	return ctx.BuildFile(expr), nil
}

func (e yamlEndec) Encode(v cue.Value) ([]byte, error) {
	s, err := pyaml.Marshal(v)
	return []byte(s), err
}
