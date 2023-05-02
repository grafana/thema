package legacy

// BindLineage takes a raw cue.Value, checks that it is a valid lineage (that it
// upholds the invariants which undergird Thema's translatability guarantees),
// and returns the cue.Value wrapped in a Lineage, iff validity checks succeed.
// The Lineage type provides access to all the types and functions for working
// with Thema in Go.
//
// This function is the sole intended mechanism for creating Lineage objects,
// thereby providing a practical promise that all instances of Lineage uphold
// Thema's invariants. It is primarily intended for use by authors of lineages
// in the creation of a LineageFactory.
// func BindLineage(raw cue.Value, rt *Runtime, opts ...BindOption) (Lineage, error) {
// 	// We could be more selective than this, but this isn't supposed to be forever, soooooo
// 	rt.l()
// 	defer rt.u()
//
// 	p := raw.Path().String()
// 	// The candidate lineage must exist.
// 	if !raw.Exists() {
// 		if p != "" {
// 			return nil, fmt.Errorf("%w: path was %q", terrors.ErrValueNotExist, p)
// 		}
//
// 		return nil, terrors.ErrValueNotExist
// 	}
// 	if p == "" {
// 		p = "instance root"
// 	}
//
// 	// The candidate lineage must be error-free.
// 	// TODO replace this with Err, this check isn't actually what we want up here. Only schemas themselves must be cycle-free
// 	if err := raw.Validate(cue.Concrete(false), cue.DisallowCycles(true)); err != nil {
// 		return nil, err
// 	}
//
// 	// The candidate lineage must be an instance of #Lineage.
// 	dlin := rt.linDef()
// 	err := dlin.Subsume(raw, cue.Raw(), cue.Schema(), cue.Final())
// 	if err != nil {
// 		// FIXME figure out how to wrap both the sentinel and CUE error sanely
// 		return nil, fmt.Errorf("%w (%s): %s", terrors.ErrValueNotALineage, p, err)
// 	}
//
// 	nam, err := raw.LookupPath(cue.MakePath(cue.Str("name"))).String()
// 	if err != nil {
// 		return nil, fmt.Errorf("%w (%s): name field is not concrete", terrors.ErrInvalidLineage, p)
// 	}
//
// 	cfg := &bindConfig{}
// 	for _, opt := range opts {
// 		opt(cfg)
// 	}
//
// 	lin := &UnaryLineage{
// 		validated: true,
// 		raw:       raw,
// 		rt:        rt,
// 		name:      nam,
// 	}
//
// 	// Populate the version list and enforce compat/subsumption invariants
// 	seqiter, _ := raw.LookupPath(cue.MakePath(cue.Str("seqs"))).List()
// 	var seqv uint
// 	var predecessor cue.Value
// 	var predsv SyntacticVersion
// 	for seqiter.Next() {
// 		var schv uint
// 		schemas := seqiter.Value().LookupPath(cue.MakePath(cue.Str("schemas")))
//
// 		schiter, _ := schemas.List()
// 		for schiter.Next() {
// 			v := synv(seqv, schv)
// 			lin.allv = append(lin.allv, v)
//
// 			sch := schiter.Value()
//
// 			defname := fmt.Sprintf("%s%v%v", util.SanitizeLabelString(nam), v[0], v[1])
// 			defpath := cue.MakePath(cue.Def(defname))
// 			defsch := rt.Context().
// 				CompileString(fmt.Sprintf("#%s: _", defname)).
// 				FillPath(defpath, sch).
// 				LookupPath(defpath)
// 			if defsch.Validate() != nil {
// 				panic(defsch.Validate())
// 			}
// 			lin.allsch = append(lin.allsch, &UnarySchema{
// 				raw:    sch,
// 				defraw: defsch,
// 				lin:    lin,
// 				v:      v,
// 			})
//
// 			// No predecessor to compare against with the very first schema
// 			if !(schv == 0 && seqv == 0) {
// 				// TODO Marked as buggy until we figure out how to both _not_ require
// 				// schema to be closed in the .cue file, _and_ how to detect default changes
// 				if !cfg.skipbuggychecks {
// 					// The sequences and schema in the candidate lineage must follow
// 					// backwards [in]compatibility rules.
// 					// TODO Subsumption may not be what we actually want to check here,
// 					// as it does not allow the addition of required fields with defaults
// 					bcompat := sch.Subsume(predecessor, cue.Raw(), cue.Schema(), cue.Definitions(true), cue.All(), cue.Final())
// 					if (schv == 0 && bcompat == nil) || (schv != 0 && bcompat != nil) {
// 						return nil, &compatInvariantError{
// 							rawlin:    raw,
// 							violation: [2]SyntacticVersion{predsv, v},
// 							detail:    bcompat,
// 						}
// 					}
// 				}
// 			}
//
// 			predecessor = sch
// 			predsv = v
// 			schv++
// 		}
// 		seqv++
// 	}
//
// 	return lin, nil
// }
