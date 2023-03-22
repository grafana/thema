package analyze

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/build"
)

// FindBuildInstance searches, recursively, for a non-nil cue.Value.BuildInstance() on:
//   - the provided raw node
//   - the first direct conjunct of the raw node, if any
//   - the ReferencePath() result, if different, of either of the above
func FindBuildInstance(raw cue.Value) *build.Instance {
	var fbi func(raw cue.Value) *build.Instance
	fbi = func(raw cue.Value) *build.Instance {
		bi := raw.BuildInstance()
		if bi != nil {
			// fmt.Println("lit")
			return bi
		}

		if ref, path := raw.ReferencePath(); len(path.Selectors()) != 0 && ref != raw {
			// if ref, path := raw.ReferencePath(); ref.Exists() && ref != raw {
			// fmt.Println("REF", ref, path)
			if bi = fbi(ref); bi != nil {
				return bi
			}
			// } else if raw.Source() != nil {
			// 	fmt.Printf("SOURCE NOT NIL %T\n", raw.Source())
			// } else {
			// 	fmt.Println("PATH", raw.Path())
		}

		// No instance on anything reachable from the value itself. Try again with any
		// immediate conjuncts.
		if op, dvals := raw.Expr(); op == cue.AndOp {
			// Only try the first value, which will represent the package. Additional values
			// will be constraints specified on the whole package instance.
			// fmt.Println("EXPR", raw, dvals)
			bi = fbi(dvals[0])
			// for _, dval := range dvals {
			// 	bi = FindBuildInstance(dval)
			// 	if bi != nil {
			// 		break
			// 	}
			// }
			// } else {
			// 	fmt.Println("OP", op)
		}
		return bi
	}

	if bi := fbi(raw); bi != nil {
		return bi
	}

	// last resort - unify the input with a dummy value to try to produce a reference
	// that will point us back to the original package
	path := cue.MakePath(cue.Str("dummy"))
	return fbi(raw.Context().CompileString("dummy: _").FillPath(path, raw))
}
