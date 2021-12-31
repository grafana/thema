package load

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/build"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/grafana/thema"
	"github.com/grafana/thema/internal/util"
)

// ErrFSNotACueModule is a general error that wraps a particular error that explains
// why a particular fs.FS cannot be used as a CUE module FS as needed by InstancesWithThema.
type ErrFSNotACueModule struct {
	fserr error
}

func (e *ErrFSNotACueModule) Error() string {
	return fmt.Sprintf("provided fs.FS is not a valid CUE module: %q", e.fserr.Error())
}

func (e *ErrFSNotACueModule) Unwrap() error {
	return e.fserr
}

var themamodpath string = filepath.Join("cue.mod", "pkg", "github.com", "grafana", "thema")

// InstancesWithThema wraps and narrows load.Instance in order to allow
// loading .cue files that directly `import "github.com/grafana/thema"`, as
// lineages are expected to. This is accomplished by constructing a
// load.Config.Overlay with the thema CUE files dynamically injected under
// cue.mod/pkg/, where CUE searches for mod-external imports.
//
// This loader is opinionated, preferring simple ease-of-use and fewer degrees
// of freedom to the completeness of load.Instances.  Some reasonable use cases
// may not be achievable. Make your own as needed - all key component parts are
// exported from elsewhere in the Thema Go module.
//
// The modFS is expected to be an fs.FS containing the cue.mod module metadata,
// and any lineage(s) to be loaded.
//
// The root of the FS must be an importable CUE module with a path. That is,
// there must exist cue.mod/module.cue, which must contain a top-level field
// declaring the module name (aka import prefix/module path), e.g.:
//
//   module: "github.com/grafana/thema"
//
// The dirs parameter should specify directories containing .cue files with
// lineages to be loaded, relative to the lmodule root directory. This is similar
// to load.Config.Dir, except:
//   - There is no corollary to the load.Config.Packages property. Consequently,
//     only .cue files with packages having the same name as their parent dir will be loaded.
//       - The package name of the root dir is the final element of the module name.
//   - Multiple directories may be loaded by passing additional strings to the
//     variadic dirs.
//   - If no values are passed for dirs, the loader will default to loading the
//     root directory of the modFS. The root directory can be explicitly specified
//     with ".".
func InstancesWithThema(ctx *build.Context, modFS fs.FS, dirs ...string) ([]*build.Instance, error) {
	var modname string
	err := fs.WalkDir(modFS, "cue.mod", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			switch path {
			case filepath.Join("cue.mod", "gen"), filepath.Join("cue.mod", "usr"):
				return fs.SkipDir
			case themamodpath:
				return fmt.Errorf("path %q already exists in modFS passed to InstancesWithThema, must be absent for dynamic dependency injection", themamodpath)
			}
			return nil
		} else if path == filepath.Join("cue.mod", "module.cue") {
			modf, err := modFS.Open(path)
			if err != nil {
				return err
			}
			b, err := io.ReadAll(modf)
			if err != nil {
				return err
			}

			modname, err = cuecontext.New().CompileBytes(b).LookupPath(cue.MakePath(cue.Str("module"))).String()
			if err != nil {
				return err
			}
			if modname == "" {
				return fmt.Errorf("InstancesWithThema requires non-empty module name in modFS' cue.mod/module.cue")
			}
		}

		return nil
	})
	if err != nil {
		return nil, &ErrFSNotACueModule{fserr: err}
	}

	if modname == "" {
		return nil, &ErrFSNotACueModule{fserr: fmt.Errorf("cue.mod/module.cue did not exist")}
	}

	if len(dirs) == 0 {
		dirs = append(dirs, ".")
	}

	modroot := filepath.FromSlash(filepath.Join(util.Prefix, modname))
	overlay := make(map[string]load.Source)
	if err := util.ToOverlay(modroot, modFS, overlay); err != nil {
		return nil, err
	}

	// Special case for when we're calling this loader with paths inside the thema module
	if modname == "github.com/grafana/thema" {
		if err := ToOverlay(modroot, thema.CueJointFS, overlay); err != nil {
			return nil, err
		}
	} else {
		if err := ToOverlay(filepath.Join(modroot, themamodpath), thema.CueFS, overlay); err != nil {
			return nil, err
		}
	}

	cfg := &load.Config{
		Overlay:    overlay,
		ModuleRoot: modroot,
		Module:     modname,
	}

	var insts []*build.Instance
	for _, dir := range dirs {
		dir = filepath.FromSlash(dir)
		if dir == "." {
			cfg.Package = filepath.Base(modroot)
			cfg.Dir = modroot
		} else {
			cfg.Dir = filepath.Join(modroot, dir)
			cfg.Package = filepath.Base(dir)
		}
		insts = append(insts, load.Instances(nil, cfg)...)
	}

	return insts, nil
}

// ToOverlay maps the provided fs.FS into an Overlay for use in load.Config.
//
// An absolute path prefix must be provided.
func ToOverlay(prefix string, vfs fs.FS, overlay map[string]load.Source) error {
	return util.ToOverlay(prefix, vfs, overlay)
}
