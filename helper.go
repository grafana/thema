package thema

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
	"fmt"
)

func printValue(v cue.Value) {
	syn := v.Syntax(
		cue.Final(),         // close structs and lists
		cue.Concrete(false), // allow incomplete values
		cue.Definitions(false),
		cue.Hidden(true),
		cue.Optional(true),
		cue.Attributes(true),
		cue.Docs(true),
	)

	// Pretty print the AST, returns ([]byte, error)
	bs, _ := format.Node(
		syn,
		format.TabIndent(true),
		// format.UseSpaces(2),
	)

	// print to stdout
	fmt.Println(string(bs))
}
