package exemplars

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing/fstest"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"github.com/grafana/thema"
	"github.com/grafana/thema/internal/util"
	tload "github.com/grafana/thema/load"
)

func buildAll(ctx *cue.Context) cue.Value {
	all, err := tload.InstancesWithThema(CueFS(), ".")
	if err != nil {
		panic(err)
	}
	return ctx.BuildInstance(all)
}

// CueFS returns an fs.FS containing the .cue files, along with a simulated
// cue.mod directory, making it suitable for use with load.InstancesWithThema().
func CueFS() fs.FS {
	m, err := populateMapFSFromRoot(cueFS, "", "")
	if err != nil {
		panic(fmt.Sprintf("broken mapfs: %s", err))
	}

	m["cue.mod/module.cue"] = &fstest.MapFile{
		Data: []byte("module: \"github.com/grafana/thema/exemplars\""),
	}
	return m
}

func populateMapFSFromRoot(in fs.FS, root, join string) (fstest.MapFS, error) {
	out := make(fstest.MapFS)
	err := fs.WalkDir(in, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Ignore gosec warning G304. The input set here is necessarily
		// constrained to files specified in embed.go
		// nolint:gosec
		b, err := os.Open(filepath.Join(root, join, path))
		if err != nil {
			return err
		}
		byt, err := io.ReadAll(b)
		if err != nil {
			return err
		}

		out[path] = &fstest.MapFile{Data: byt}
		return nil
	})
	return out, err
}

// NarrowingLineage returns a handle for using the "narrowing" exemplar lineage.
func NarrowingLineage(lib thema.Library, o ...thema.BindOption) (thema.Lineage, error) {
	return lineageForExemplar("narrowing", lib, o...)
}

// RenameLineage returns a handle for using the "Rename" exemplar lineage.
func RenameLineage(lib thema.Library, o ...thema.BindOption) (thema.Lineage, error) {
	return lineageForExemplar("rename", lib, o...)
}

// DefaultChangeLineage returns a handle for using the "defaultchange" exemplar lineage.
func DefaultChangeLineage(lib thema.Library, o ...thema.BindOption) (thema.Lineage, error) {
	return lineageForExemplar("defaultchange", lib, o...)
}

// ExpandLineage returns a handle for using the "expand" exemplar lineage.
func ExpandLineage(lib thema.Library, o ...thema.BindOption) (thema.Lineage, error) {
	return lineageForExemplar("expand", lib, o...)
}

// SingleLineage returns a handle for using the "single" exemplar lineage.
func SingleLineage(lib thema.Library, o ...thema.BindOption) (thema.Lineage, error) {
	return lineageForExemplar("single", lib, o...)
}

var _ thema.LineageFactory = NarrowingLineage
var _ thema.LineageFactory = RenameLineage
var _ thema.LineageFactory = DefaultChangeLineage
var _ thema.LineageFactory = ExpandLineage
var _ thema.LineageFactory = SingleLineage

// Build the harness containing a single exemplar lineage.
func harnessForExemplar(name string, lib thema.Library) cue.Value {
	all := buildExemplarsPackage(lib)

	lval := all.LookupPath(cue.MakePath(cue.Str(name)))
	if !lval.Exists() {
		panic(fmt.Sprintf("no exemplar exists with name %q", name))
	}

	return lval
}

// Build a Lineage representing a single exemplar.
func lineageForExemplar(name string, lib thema.Library, o ...thema.BindOption) (thema.Lineage, error) {
	return thema.BindLineage(harnessForExemplar(name, lib), lib, o...)
}

func buildExemplarsPackage(lib thema.Library) cue.Value {
	ctx := lib.UnwrapCUE().Context()

	overlay, err := exemplarOverlay()
	if err != nil {
		panic(err)
	}

	cfg := &load.Config{
		Overlay: overlay,
		Module:  "github.com/grafana/thema",
		Dir:     filepath.Join(util.Prefix, "exemplars"),
	}

	return ctx.BuildInstance(load.Instances(nil, cfg)[0])
}

func exemplarOverlay() (map[string]load.Source, error) {
	overlay := make(map[string]load.Source)

	if err := util.ToOverlay(util.Prefix, thema.CueJointFS, overlay); err != nil {
		return nil, err
	}

	if err := util.ToOverlay(filepath.Join(util.Prefix, "exemplars"), cueFS, overlay); err != nil {
		return nil, err
	}

	return overlay, nil
}
