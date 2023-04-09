package thema

import (
	"fmt"

	"cuelang.org/go/cue"
	cerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/token"
	"github.com/cockroachdb/errors"
	terrors "github.com/grafana/thema/errors"
	"github.com/grafana/thema/internal/compat"
)

// maybeLineage is an intermediate processing structure used to validate
// inputs as actual lineages
//
// it's important that these flags are populated in order to avoid false negatives.
// no system ensures this, it's all human reasoning
type maybeLineage struct {
	// original raw input cue.Value
	raw cue.Value

	// input cue.Value, unified with thema.#Lineage
	uni cue.Value

	rt *Runtime

	// pos of the original input for the lineage
	pos token.Pos

	// bind options passed by the caller
	cfg *bindConfig

	schlist []*schemaDef

	allv []SyntacticVersion

	// thema.#Lineage was unified with the raw input prior to it being passed to
	// BindLineage
	// rawUnifiesLineage bool

	// The raw input value is the root of a package instance
	// rawIsPackage bool
}

func (ml *maybeLineage) checkGoValidity() error {
	schiter, err := ml.uni.LookupPath(cue.MakePath(cue.Hid("_sortedSchemas", "github.com/grafana/thema"))).List()
	if err != nil {
		panic(fmt.Sprintf("unreachable - should have already verified schemas field exists and is list: %+v", cerrors.Details(err, nil)))
	}
	schpath := cue.MakePath(cue.Hid("_#schema", "github.com/grafana/thema"))
	vpath := cue.MakePath(cue.Str("version"))

	var previous *schemaDef
	for schiter.Next() {
		// Only thing not natively enforced in CUE is that the #SchemaDef.version field is concrete
		svval := schiter.Value().LookupPath(vpath)
		iter, err := svval.List()
		if err != nil {
			panic(fmt.Sprintf("unreachable - should have already verified #SchemaDef.version field exists and is list: %+v", err))
		}
		for iter.Next() {
			if !iter.Value().IsConcrete() {
				return errors.Mark(mkerror(iter.Value(), "#SchemaDef.version must have concrete major and minor versions"), terrors.ErrInvalidLineage)
			}
		}
		sch := &schemaDef{}
		err = svval.Decode(&sch.v)
		if err != nil {
			panic(fmt.Sprintf("unreachable - could not decode syntactic version: %+v", err))
		}

		sch.ref = schiter.Value()
		sch.def = sch.ref.LookupPath(schpath)
		if previous != nil {
			compaterr := compat.ThemaCompatible(sch.def, previous.def)
			if (sch.v[1] == 0 && compaterr == nil) || (sch.v[1] != 0 && compaterr != nil) {
				return &compatInvariantError{
					rawlin:    ml.uni,
					violation: [2]SyntacticVersion{previous.v, sch.v},
					detail:    compaterr,
				}
			}
		}

		ml.schlist = append(ml.schlist, sch)
		ml.allv = append(ml.allv, sch.v)
		previous = sch
	}

	return nil
}

func (ml *maybeLineage) checkExists() error {
	p := ml.raw.Path().String()
	// The candidate lineage must exist.
	// TODO can we do any better with contextualizing these errors?
	if !ml.raw.Exists() {
		if p != "" {
			return errors.Mark(errors.Newf("not a lineage: no cue value at path %q", p), terrors.ErrValueNotExist)
		}

		return errors.WithStack(terrors.ErrValueNotExist)
	}
	return nil
}

func (ml *maybeLineage) checkLineageShape() error {
	// Check certain paths specifically, because these are common getting started errors of just arranging
	// CUE statements in the right way that deserve more targeted guidance
	for _, path := range []string{"name", "schemas"} {
		val := ml.raw.LookupPath(cue.MakePath(cue.Str(path)))
		if !val.Exists() {
			return errors.Mark(mkerror(ml.raw, "not a lineage, missing #Lineage.%s", path), terrors.ErrValueNotALineage)
		}
		if !val.IsConcrete() {
			return errors.Mark(mkerror(val, "invalid lineage, #Lineage.%s must be concrete", path), terrors.ErrInvalidLineage)
		}
	}

	// The candidate lineage must be an instance of #Lineage. However, we can't validate the whole
	// structure, because lenses will fail validation. This is because we currently expect them to be written:
	//
	// {
	// 		input: _
	// 		result: {
	// 			foo: input.foo
	// 		}
	// }
	//
	// means that those structures won't pass Validate until we've injected an actual object there.
	if err := ml.uni.Validate(cue.Final()); err != nil {
		return errors.Mark(cerrors.Promote(err, "not an instance of thema.#Lineage"), terrors.ErrInvalidLineage)
	}

	return nil
}

// Checks the validity properties of lineages that are expressible natively in CUE.
func (ml *maybeLineage) checkNativeValidity() error {
	// The candidate lineage must be error-free.
	// TODO replace this with Err, this check isn't actually what we want up here. Only schemas themselves must be cycle-free
	if err := ml.raw.Validate(cue.Concrete(false)); err != nil {
		return errors.Mark(cerrors.Promote(err, "lineage is invalid"), terrors.ErrInvalidLineage)
	}
	if err := ml.uni.Validate(cue.Concrete(false)); err != nil {
		return errors.Mark(cerrors.Promote(err, "lineage is invalid"), terrors.ErrInvalidLineage)
	}

	return nil
}
