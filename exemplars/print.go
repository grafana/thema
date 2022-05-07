package thema

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
)

func dumpLineage(lin Lineage) ([]byte, error) {
	v := lin.UnwrapCUE()
	syn := v.Syntax(
		cue.Definitions(true),
		cue.Hidden(true),
		cue.Optional(true),
		cue.Attributes(true),
		cue.Docs(true),
	)

	return format.Node(syn, format.TabIndent(true))
}
