package main

import (
	"fmt"
	"os"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/build"
	"cuelang.org/go/cue/load"
	"github.com/grafana/thema"
)

// lineageFromPaths takes a filepath and an optional CUE path expression
// and loads the result up and bind it to a Lineage.
func lineageFromPaths(lib thema.Library, filepath, cuepath string) (thema.Lineage, error) {
	if filepath == "" {
		panic("empty filepath")
	}

	info, err := os.Stat(filepath)
	if err != nil {
		return nil, err
	}

	binsts := load.Instances([]string{filepath}, &load.Config{})
	return buildInsts(lib, binsts, func(binst *build.Instance) string {
		if info.IsDir() {
			return fmt.Sprintf("%s:%s", filepath, binst.PkgName)
		}
		return filepath
	}, cuepath)
}

func lineageFromStdin(lib thema.Library, b []byte, cuepath string) (thema.Lineage, error) {
	overlay := map[string]load.Source{
		"stdin": load.FromBytes(b),
	}

	cfg := &load.Config{
		Overlay: overlay,
	}

	binsts := load.Instances([]string{"stdin"}, cfg)
	return buildInsts(lib, binsts, func(binst *build.Instance) string {
		return "stdin"
	}, cuepath)
}

type ppathf func(*build.Instance) string

func buildInsts(lib thema.Library, binsts []*build.Instance, ppath ppathf, cuepath string) (thema.Lineage, error) {
	rets := make([]struct {
		lin thema.Lineage
		err error
	}, len(binsts))
	for i, binst := range binsts {
		rets[i].lin, rets[i].err = loadone(lib, binst, ppath(binst), cuepath)
	}

	switch len(binsts) {
	case 0:
		// TODO better error - ugh i wish CUE's docs made the failure modes here clearer
		return nil, fmt.Errorf("no loadable CUE data found")
	case 1:
		return rets[0].lin, rets[0].err
	default:
		// Try all of them. Error if we end up with more than one.
		var lin thema.Lineage
		for _, ret := range rets {
			if ret.lin != nil {
				if lin != nil {
					return nil, fmt.Errorf("valid lineages found in multiple CUE packages")
				}
				lin = ret.lin
			}
		}

		if lin == nil {
			// Sloppy, but it's almost always gonna be the first one
			return nil, rets[0].err
		}
		return lin, nil
	}
}

func loadone(lib thema.Library, binst *build.Instance, pkgpath, cuepath string) (thema.Lineage, error) {
	if binst.Err != nil {
		return nil, binst.Err
	}

	v := lib.UnwrapCUE().Context().BuildInstance(binst)
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
	// FIXME so hacky to write back to a global this way - only OK because buildInsts guarantees only one can escape
	linbinst = binst

	return thema.BindLineage(v, lib)
}
