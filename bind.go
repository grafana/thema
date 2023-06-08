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
	// user lineage definition, NOT unified with thema.#Lineage
	raw cue.Value

	// user lineage definition, unified with thema.#Lineage
	uni cue.Value

	// original input cue.Value representing the lineage. May or may not be unified
	// with thema.#Lineage
	orig cue.Value

	rt *Runtime

	// pos of the original input for the lineage
	pos token.Pos

	// bind options passed by the caller
	cfg *bindConfig

	schlist []*schemaDef

	allv []SyntacticVersion

	// The raw input value is the root of a package instance
	// rawIsPackage bool
}

func (ml *maybeLineage) checkGoValidity(cfg *bindConfig) error {
	// schiter, err := ml.uni.LookupPath(cue.MakePath(cue.Hid("_sortedSchemas", "github.com/grafana/thema"))).List()
	schiter, err := ml.uni.LookupPath(cue.MakePath(cue.Str("schemas"))).List()
	if err != nil {
		panic(fmt.Sprintf("unreachable - should have already verified schemas field exists and is list: %+v", cerrors.Details(err, nil)))
	}
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
		sch.def = sch.ref.LookupPath(pathSchDef)
		if previous != nil && !cfg.skipbuggychecks {
			compaterr := compat.ThemaCompatible(previous.ref.LookupPath(pathSch), sch.ref.LookupPath(pathSch))
			if sch.v[1] == 0 && compaterr == nil {
				// Major version change, should be backwards incompatible
				return errors.Mark(mkerror(sch.ref.LookupPath(pathSch), "schema %s must be backwards incompatible with schema %s: introduce a breaking change, or redeclare as version %s", sch.v, previous.v, synv(previous.v[0], previous.v[1]+1)), terrors.ErrInvalidLineage)
			}
			if sch.v[1] != 0 && compaterr != nil {
				// Minor version change, should be backwards compatible
				return errors.Mark(mkerror(sch.ref.LookupPath(pathSch), "schema %s is not backwards compatible with schema %s:\n%s", sch.v, previous.v, cerrors.Details(compaterr, nil)), terrors.ErrInvalidLineage)
			}
		}

		ml.schlist = append(ml.schlist, sch)
		ml.allv = append(ml.allv, sch.v)
		previous = sch
	}

	return nil
}

func (ml *maybeLineage) checkExists(cfg *bindConfig) error {
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

func (ml *maybeLineage) checkLineageShape(cfg *bindConfig) error {
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
func (ml *maybeLineage) checkNativeValidity(cfg *bindConfig) error {
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
