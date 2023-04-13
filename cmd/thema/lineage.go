package main

import (
	"fmt"

	upcue "cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/ast/astutil"
	tastutil "github.com/grafana/thema/internal/astutil"
	"github.com/spf13/cobra"
)

var linCmd = &cobra.Command{
	Use:   "lineage <command>",
	Short: "Perform operations directly on lineages and schema",
	Long: `Perform operations directly on lineages and schema.

Create, modify, and validate Thema lineages. Generate representations
of Thema lineages and schema in other schema languages.
`,
}

func setupLineageCommand(cmd *cobra.Command) {
	cmd.AddCommand(linCmd)
	ic := new(initCommand)
	ic.setup(linCmd)

	bc := new(bumpCommand)
	bc.setup(linCmd)

	gc := new(genCommand)
	gc.setup(linCmd)

	fc := new(fixCommand)
	fc.setup(linCmd)
}

func toSubpath(subpath string, f *ast.File) (*ast.File, error) {
	if subpath == "" {
		return f, nil
	}

	p := upcue.ParsePath(subpath)
	if p.Err() != nil {
		return nil, fmt.Errorf("invalid path provided for --cue-path: %w", p.Err())
	}

	err := astutil.Sanitize(f)
	if err != nil {
		return nil, fmt.Errorf("error while sanitizing generated file: %w", err)
	}

	// in := ctx.BuildFile(f)
	// if in.Err() != nil {
	// 	return nil, fmt.Errorf("error when building value: %w", in.Err())
	// }

	out := empt().FillPath(p, empt())
	if out.Err() != nil {
		return nil, fmt.Errorf("error in lineage when placed at requested --cue-path: %w", out.Err())
	}

	var nf *ast.File
	switch x := tastutil.Format(out).(type) {
	case *ast.File:
		nf = x
	case ast.Expr:
		nf, err = astutil.ToFile(x)
		if err != nil {
			return nil, fmt.Errorf("error converting expr to file: %w", err)
		}
	}

	var done bool
	astutil.Apply(nf, func(c astutil.Cursor) bool {
		if x, ok := c.Node().(*ast.StructLit); ok {
			if len(x.Elts) == 0 {
				x.Elts = f.Decls
				done = true
			}
		}
		return !done
	}, nil)

	f.Decls = nf.Decls
	return f, nil
}
