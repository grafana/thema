package kernel

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/encoding/json"
)

// A DataLoader takes some input data as a []byte and loads it into the provided
// cue.Context, and returns a handle to the corresponding cue.Value.
type DataLoader func(*cue.Context, []byte) (cue.Value, error)

// NewJSONDecoder creates a DataLoader func that translates a []byte containing
// JSON data into a cue.Value.
//
// The provided path is used as the sourcename for the input data (the
// identifier for the data used by CUE error messages). Any provided
// cue.BuildOptions are passed along to cue.Context.BuildExpr().
func NewJSONDecoder(path string, o ...cue.BuildOption) DataLoader {
	return func(ctx *cue.Context, data []byte) (cue.Value, error) {
		expr, err := json.Extract(path, data)
		if err != nil {
			return cue.Value{}, err
		}
		return ctx.BuildExpr(expr, o...), nil
	}
}
