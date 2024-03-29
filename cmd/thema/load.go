package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/build"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/grafana/thema"
	terrors "github.com/grafana/thema/errors"
	"github.com/grafana/thema/internal/util"
	"github.com/spf13/cobra"
)

// TODO OMG WOULD BE SO AMAZING TO MAKE THIS ALL JUST USE github.com/grafana/thema/load

type lineageLoadArgs struct {
	// fs path to the lineage to load. may be absolute or relative, and point to a
	// file or dir. Also may be "-" for stdin
	inputLinFilePath string

	// Absolute path version of inputLinFilePath
	absInput string

	// indicates whether inputLinFilePath points to a directory. Populated in
	// dynLoad
	pathIsDir bool

	// cue path to the lineage to work on (default root)
	lincuepath string

	// String argument of a version
	verstr string

	// skip binding the lineage - useful when operating on values that are known to be
	// invalid lineages
	skipBindLineage bool

	// only do all the loading once
	once sync.Once

	// Stores all results of running loaders/validators
	dl *dynamicLoader

	dlerr error
}

type dynamicLoader struct {
	// if a real cue.mod existed on disk, this is non-nil
	cm *parentCueMod
	// relative path from the cue mod to the dir containing the loaded lineage, if there was a cue.mod
	relToLinDir string
	// filename of the lineage, if a single specific filename was provided
	linFilename string

	// build.Instance from which the lineage came
	binst *build.Instance
	// loaded lineage
	lin thema.Lineage

	// loaded schema. latest if no version was provided. it's up to the command
	// to decide if it's acceptable to use latest - this gets loaded either way.
	sch thema.Schema
}

// lineageFromPaths takes a filepath and an optional CUE path expression
// and loads the result up and bind it to a Lineage.
type parentCueMod struct {
	// the absolute path to the dir containing cue.mod directory
	cuemodparentdir string
	// name of module according to cue.mod/module.cue file
	modname string
}

func (lla *lineageLoadArgs) dynLoad() (*dynamicLoader, error) {
	if lla.inputLinFilePath == "" {
		return nil, errors.New("must provide a lineage file argument via -l")
	}

	dl := &dynamicLoader{}
	// scenarios
	// - lineage load path has no cue.mod parent (dynamically create cueFS in load dir)
	// - lineage load path has a cue.mod in some parent (dynamically make a cueFS that includes it; record rel)
	// - cwd is in the same cue.mod as target lineage
	// - cwd is different
	//
	// loadpath is absolute vs. relative (CUE load.Instances() seems to choke on absolute paths)

	var err error
	lla.absInput, err = filepath.Abs(lla.inputLinFilePath)
	if err != nil {
		return nil, fmt.Errorf("error getting absolute filepath for %s: %w", lla.inputLinFilePath, err)
	}

	var binsts []*build.Instance
	var cfg *load.Config

	// GOAL:
	// cfg.ModuleRoot is the ABSOLUTE path up to the module root, real or virtual
	// cfg.Module is the name of the module, real or virtual
	// cfg.Dir is the RELATIVE path, from module root to what we actually want to load, less filename

	var info os.FileInfo
	info, err = os.Stat(lla.absInput)
	if err != nil {
		return nil, fmt.Errorf("error statting path %q: %w", lla.absInput, err)
	}
	lla.pathIsDir = info.IsDir()
	dl.cm, err = findCueMod(lla.absInput)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("error while searching for cue.mod at or above %s: %w", lla.inputLinFilePath, err)
		}

		dyndir := lla.absInput
		var args []string
		if !lla.pathIsDir {
			args = append(args, filepath.Base(dyndir))
			dyndir = filepath.Dir(dyndir)
		}

		overlay := map[string]load.Source{
			filepath.Join(dyndir, "cue.mod", "module.cue"): load.FromString(`module: ""`),
		}
		if err := util.ToOverlay(filepath.Join(dyndir, themamodpath), thema.CueFS, overlay); err != nil {
			panic(fmt.Sprintf("unreachable - %s", err))
		}
		cfg = &load.Config{
			ModuleRoot: dyndir,
			Overlay:    overlay,
			Dir:        dyndir,
			Package:    "*",
		}

		binsts = load.Instances(args, cfg)
	} else {
		overlay := map[string]load.Source{}
		// do no muckery if in thema mod dir
		if dl.cm.modname != "github.com/grafana/thema" {
			if err := util.ToOverlay(filepath.Join(dl.cm.cuemodparentdir, themamodpath), thema.CueFS, overlay); err != nil {
				panic(fmt.Sprintf("unreachable - %s", err))
			}
		}

		dl.relToLinDir, err = filepath.Rel(dl.cm.cuemodparentdir, lla.absInput)
		if err != nil {
			return nil, fmt.Errorf("should be unreachable - cue.mod 'parent' path of %s is not rel to lin filepath of %s", dl.cm.cuemodparentdir, lla.inputLinFilePath)
		}
		var args []string
		if !lla.pathIsDir {
			dl.linFilename = filepath.Base(dl.relToLinDir)
			dl.relToLinDir = filepath.Dir(dl.relToLinDir)
			args = append(args, dl.linFilename)
		}

		cfg = &load.Config{
			ModuleRoot: dl.cm.cuemodparentdir,
			Module:     dl.cm.modname,
			Overlay:    overlay,
			Dir:        filepath.Join(dl.cm.cuemodparentdir, dl.relToLinDir),
			// Package:    "*",
		}
		binsts = load.Instances(args, cfg)
	}

	if lla.skipBindLineage {
		// just take the first one, ooof
		dl.binst = binsts[0]
		return dl, nil
	}

	dl.lin, dl.binst, err = buildInsts(rt, binsts, func(binst *build.Instance) string {
		if lla.pathIsDir {
			return fmt.Sprintf("%s:%s", lla.inputLinFilePath, binst.PkgName)
		}
		return lla.inputLinFilePath
	}, lla.lincuepath)
	if err != nil {
		return nil, err
	}

	// Now attach the schema - other validators can decide if what we loaded here
	// was OK (i.e. if command required explicit input)
	if lla.verstr == "" {
		dl.sch = dl.lin.Latest()
	} else {
		synv, err := thema.ParseSyntacticVersion(lla.verstr)
		if err != nil {
			return nil, err
		}
		dl.sch, err = dl.lin.Schema(synv)
		if err != nil {
			return nil, fmt.Errorf("schema version %v does not exist in lineage", synv)
		}
	}

	return dl, nil
}

func (lla *lineageLoadArgs) dynLoadOnce() error {
	lla.once.Do(func() {
		lla.dl, lla.dlerr = lla.dynLoad()
	})
	return lla.dlerr
}

func (lla *lineageLoadArgs) validateLineageInput(cmd *cobra.Command, args []string) error {
	if err := lla.dynLoadOnce(); err != nil {
		if errors.Is(err, terrors.ErrValueNotALineage) && strings.Contains(err.Error(), "instance root") {
			return fmt.Errorf("%w\nDid you forget to pass a CUE path with -p?", err)
		}
		return err
	}
	return nil
}

func (lla *lineageLoadArgs) validateVersionInput(cmd *cobra.Command, args []string) error {
	if err := lla.dynLoadOnce(); err != nil {
		return err
	}
	if lla.verstr == "" {
		return fmt.Errorf("must specify an explicit schema version")
	}
	return nil
}

func (lla *lineageLoadArgs) validateVersionInputOptional(cmd *cobra.Command, args []string) error {
	return lla.dynLoadOnce()
}

// findCueMod recursively searches the given path and its parent directories
// until one is found containing a cue.mod/module.cue. An error is returned on
// file read errors, or if no cue.mod was found.
func findCueMod(path string) (*parentCueMod, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("could not stat path: %w", err)
	}
	if !stat.IsDir() {
		path = filepath.Dir(path)
	}
	p := path

	for ; p != filepath.Dir(p); p = filepath.Dir(p) {
		mpath := filepath.Join(p, "cue.mod", "module.cue")
		byt, err := os.ReadFile(mpath)
		if err == nil {
			modname, err := cuecontext.New().CompileBytes(byt).LookupPath(cue.MakePath(cue.Str("module"))).String()
			if err != nil {
				return nil, fmt.Errorf("contents of %s invalid: %w", mpath, err)
			}
			return &parentCueMod{
				cuemodparentdir: p,
				modname:         modname,
			}, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("error reading path %s: %w", mpath, err)
		}
	}
	return nil, os.ErrNotExist
}

var themamodpath = filepath.Join("cue.mod", "pkg", "github.com", "grafana", "thema")

type ppathf func(*build.Instance) string

func buildInsts(rt *thema.Runtime, binsts []*build.Instance, ppath ppathf, cuepath string) (thema.Lineage, *build.Instance, error) {
	rets := make([]struct {
		lin   thema.Lineage
		binst *build.Instance
		err   error
	}, len(binsts))
	for i, binst := range binsts {
		rets[i].binst = binst
		rets[i].lin, rets[i].err = loadone(rt, binst, ppath(binst), cuepath)
	}

	switch len(binsts) {
	case 0:
		// TODO better error - ugh i wish CUE's docs made the failure modes here clearer
		return nil, nil, fmt.Errorf("no loadable CUE data found")
	case 1:
		return rets[0].lin, rets[0].binst, rets[0].err
	default:
		// Try all of them. Error if we end up with more than one.
		var lin thema.Lineage
		var binst *build.Instance
		for _, ret := range rets {
			if ret.lin != nil {
				if lin != nil {
					return nil, nil, fmt.Errorf("valid lineages found in multiple CUE packages")
				}
				lin, binst = ret.lin, ret.binst
			}
		}

		if lin == nil {
			// Sloppy, but it's almost always gonna be the first one
			return nil, nil, rets[0].err
		}
		return lin, binst, nil
	}
}

func loadone(rt *thema.Runtime, binst *build.Instance, pkgpath, cuepath string) (thema.Lineage, error) {
	if binst.Err != nil {
		return nil, binst.Err
	}

	v := rt.Underlying().Context().BuildInstance(binst)
	if !v.Exists() {
		return nil, fmt.Errorf("empty instance at %s", pkgpath)
	}

	if cuepath != "" {
		p := cue.ParsePath(cuepath)
		if p.Err() != nil {
			return nil, fmt.Errorf("%q is not a valid CUE path expression: %s", cuepath, p.Err())
		}
		v = v.LookupPath(p)
		if !v.Exists() {
			return nil, fmt.Errorf("no value at path %q in instance %q", cuepath, pkgpath)
		}
	}

	var opts []thema.BindOption
	if _, set := os.LookupEnv("THEMA_SKIP_BUGGY"); set {
		opts = append(opts, thema.SkipBuggyChecks())
	}
	return thema.BindLineage(v, rt, opts...)
}
