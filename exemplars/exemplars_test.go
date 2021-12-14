package exemplars

import (
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
	"github.com/grafana/scuemata"
	"github.com/grafana/scuemata/internal/util"
)

func TestExemplarValidity(t *testing.T) {
	overlay, err := exemplarOverlay()
	if err != nil {
		t.Fatal(err)
	}

	ctx := cuecontext.New()
	cfg := &load.Config{
		Overlay: overlay,
		Module:  "github.com/grafana/scuemata",
		Dir:     filepath.Join(util.Prefix, "exemplars"),
	}

	all := ctx.BuildInstance(load.Instances(nil, cfg)[0])

	iter, err := all.Fields(cue.Definitions(false))
	if err != nil {
		t.Fatal(err)
	}

	lib := scuemata.NewLibrary(ctx)
	for iter.Next() {
		lin := iter.Value().LookupPath(cue.ParsePath("l"))
		name, _ := lin.LookupPath(cue.ParsePath("Name")).String()
		t.Run(name, func(t *testing.T) {
			switch name {
			case "defaultchange", "narrowing", "rename":
				// subsumption in cue v0.4.0 panics in all three of these cases
				t.Skip()
			}
			err = scuemata.ValidateCompatibilityInvariants(lin, lib)
			if err != nil {
				t.Fatal(errors.Details(err, nil))
			}
		})
	}
}

func exemplarOverlay() (map[string]load.Source, error) {
	overlay := make(map[string]load.Source)

	if err := util.ToOverlay(util.Prefix, scuemata.CueJointFS, overlay); err != nil {
		return nil, err
	}

	if err := util.ToOverlay(filepath.Join(util.Prefix, "exemplars"), CueFS, overlay); err != nil {
		return nil, err
	}

	return overlay, nil
}