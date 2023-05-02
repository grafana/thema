package utils

import (
	"testing/fstest"

	"cuelang.org/go/cue/cuecontext"
	"github.com/grafana/thema"
	"github.com/grafana/thema/load"
)

// GetLineageFromBytes takes bytes from a CUE file and return an instance of thema.Lineage
// TODO: this should probably go somewhere else with less hardcoded random values?
func GetLineageFromBytes(lineage []byte) (thema.Lineage, error) {
	fs := fstest.MapFS{
		"tmp.cue": &fstest.MapFile{
			Data: append([]byte("package schemaregistry\n"), lineage...),
		},
		"cue.mod/module.cue": &fstest.MapFile{
			Data: []byte("module: \"thema.test/schemaregistry\""),
		},
	}

	inst, err := load.InstanceWithThema(fs, ".")
	if err != nil {
		return nil, err
	}

	rt := thema.NewRuntime(cuecontext.New())
	val := rt.Context().BuildInstance(inst)
	if err = val.Err(); err != nil {
		return nil, err
	}

	return thema.BindLineage(val, rt)
}
