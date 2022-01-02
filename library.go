package thema

import (
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"github.com/grafana/thema/internal/util"
)

// Library is a gateway to the set of CUE constructs available in the thema
// CUE package, allowing Go code to rely on the same functionality.
//
// Each Library is bound to a single cue.Context (Runtime), set at the time
// of Library creation via NewLibrary.
type Library struct {
	val cue.Value
}

// NewLibrary parses, loads and builds a full CUE instance/value representing
// all of the logic in the thema CUE package (github.com/grafana/thema),
// and returns a Library instance ready for use.
//
// Building is performed using the provided cue.Context. Passing a nil context will panic.
//
// This function is the canonical way to make thema logic callable from Go code.
func NewLibrary(ctx *cue.Context) Library {
	if ctx == nil {
		panic("nil context provided")
	}

	path := filepath.Join(util.Prefix, "github.com", "grafana", "thema")

	overlay := make(map[string]load.Source)
	if err := util.ToOverlay(path, CueJointFS, overlay); err != nil {
		// It's impossible for this to fail barring temporary bugs in filesystem
		// layout within the thema lib itself. These should be trivially
		// catchable during CI, so avoid forcing meaningless error handling on
		// dependers and prefer a panic.
		panic(err)
	}

	cfg := &load.Config{
		Overlay: overlay,
		Package: "thema",
		Module:  "github.com/grafana/thema",
		Dir:     path,
	}

	lib := ctx.BuildInstance(load.Instances(nil, cfg)[0])
	if lib.Err() != nil {
		// As with the above, an error means that a problem exists in the
		// literal CUE code embedded in this version of package (that should
		// have trivially been caught with CI), so the caller can't fix anything
		// without changing the version of the thema Go library they're
		// depending on. It's a hard failure that should be unreachable outside
		// thema internal testing, so just panic.
		panic(lib.Err())
	}

	return Library{
		val: lib,
	}
}

func (lib Library) RawValue() cue.Value {
	return lib.val
}

func (lib Library) Context() *cue.Context {
	return lib.val.Context()
}

// Return the #Lineage definition (or panic)
func (lib Library) linDef() cue.Value {
	dlin := lib.val.LookupPath(cue.MakePath(cue.Def("#Lineage")))
	if dlin.Err() != nil {
		panic(dlin.Err())
	}
	return dlin
}
