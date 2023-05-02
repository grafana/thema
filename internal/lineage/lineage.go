package lineage

import (
	"testing/fstest"

	"cuelang.org/go/cue/cuecontext"
	"github.com/grafana/thema"
	"github.com/grafana/thema/load"
)

// GetLineageFromBytes takes bytes from a CUE file and return an instance of thema.Lineage
func GetLineageFromBytes(lineage []byte) (thema.Lineage, error) {
	fs := fstest.MapFS{
		"tmp.cue": &fstest.MapFile{
			Data: append([]byte("package tmp\n"), lineage...),
		},
		"cue.mod/module.cue": &fstest.MapFile{
			Data: []byte("module: \"thema.lineage/tmp\""),
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
