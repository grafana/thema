package thema

import (
	"fmt"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
	"github.com/grafana/thema/internal/util"
)

// Library is a gateway to the set of CUE constructs available in the thema
// CUE package, allowing Go code to rely on the same functionality.
//
// Each Library is bound to a single cue.Context (Runtime), set at the time
// of Library creation via NewLibrary.
type Library struct {
	// Value corresponds to loading the whole github.com/grafana/thema:thema
	// package.
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
	if lib.Validate(cue.All()) != nil {
		// As with the above, an error means that a problem exists in the
		// literal CUE code embedded in this version of package (that should
		// have trivially been caught with CI), so the caller can't fix anything
		// without changing the version of the thema Go library they're
		// depending on. It's a hard failure that should be unreachable outside
		// thema internal testing, so just panic.
		panic(lib.Validate(cue.All()))
	}

	return Library{
		val: lib,
	}
}

// UnwrapCUE returns the underlying cue.Value representing the whole Thema CUE
// library (github.com/grafana/thema).
func (lib Library) UnwrapCUE() cue.Value {
	return lib.val
}

// Context returns the *cue.Context in which this library was built.
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

type cueArgs map[string]interface{}

func (ca cueArgs) make(path string, lib Library) (cue.Value, error) {
	var cpath cue.Path
	if path[0] == '_' {
		cpath = cue.MakePath(cue.Hid(path, "github.com/grafana/thema"))
	} else {
		cpath = cue.ParsePath(path)
	}
	cfunc := lib.val.LookupPath(cpath)
	if !cfunc.Exists() {
		panic(fmt.Sprintf("cannot call nonexistent CUE func %q", path))
	}
	if cfunc.Err() != nil {
		panic(cfunc.Err())
	}

	applic := []cue.Value{cfunc}
	for arg, val := range ca {
		p := cue.ParsePath(arg)
		step := applic[len(applic)-1]
		if !step.Allows(p.Selectors()[0]) {
			panic(fmt.Sprintf("CUE func %q does not take an argument named %q", path, arg))
		}
		applic = append(applic, step.FillPath(p, val))
	}
	last := applic[len(applic)-1]

	// Have to do the error check in a separate loop after all args are applied,
	// because args may depend on each other and erroneously error depending on
	// the order of application.
	for arg := range ca {
		argv := last.LookupPath(cue.ParsePath(arg))
		if argv.Err() != nil {
			return cue.Value{}, &errInvalidCUEFuncArg{
				cuefunc: path,
				argpath: arg,
				err:     argv.Err(),
			}
		}
	}
	return last, nil
}

func (ca cueArgs) call(path string, lib Library) (cue.Value, error) {
	v, err := ca.make(path, lib)
	if err != nil {
		return cue.Value{}, err
	}
	return v.LookupPath(outpath), nil
}

type errInvalidCUEFuncArg struct {
	cuefunc string
	argpath string
	err     error
}

func (e *errInvalidCUEFuncArg) Error() string {
	return fmt.Sprintf("err on arg %q to CUE func %q: %s", e.argpath, e.cuefunc, errors.Details(e.err, nil))
}

var outpath = cue.MakePath(cue.Str("out"))
