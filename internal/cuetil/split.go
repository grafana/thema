package cuetil

import (
	"fmt"

	"cuelang.org/go/cue"
)

// AppendSplit recursively splits an expression in a single cue.Value by a
// single operation, flattening it into the slice of cue.Value that
// are joined by the provided operation in the input value.
//
// Most calls to this should pass nil for the third parameter.
func AppendSplit(v cue.Value, splitBy cue.Op, a []cue.Value) []cue.Value {
	op, args := v.Expr()
	// dedup elements.
	k := 1
outer:
	for i := 1; i < len(args); i++ {
		for j := 0; j < k; j++ {
			if args[i].Subsume(args[j], cue.Raw()) == nil &&
				args[j].Subsume(args[i], cue.Raw()) == nil {
				continue outer
			}
		}
		args[k] = args[i]
		k++
	}
	args = args[:k]

	if op == cue.NoOp && len(args) == 1 {
		// TODO: this is to deal with default value removal. This may change
		// when we completely separate default values from values.
		a = append(a, args...)
	} else if op != splitBy {
		a = append(a, v)
	} else {
		for _, v := range args {
			a = AppendSplit(v, splitBy, a)
		}
	}
	return a
}

// PrintPosList dumps the cue.Value.Pos for each unified element of the provided
// cue.Value.
//
// Useful for debugging values with complex multiple unified antecedents.
func PrintPosList(v cue.Value) {
	for i, dval := range AppendSplit(v, cue.AndOp, nil) {
		fmt.Println(i, dval.Pos())
	}
}
