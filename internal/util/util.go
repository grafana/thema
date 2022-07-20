package util

import (
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/ast/astutil"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/cue/parser"
	tastutil "github.com/grafana/thema/internal/astutil"
)

// ToOverlay converts an fs.FS into a CUE loader overlay.
func ToOverlay(prefix string, vfs fs.FS, overlay map[string]load.Source) error {
	// TODO why not just stick the prefix on automatically...?
	if !filepath.IsAbs(prefix) {
		return fmt.Errorf("must provide absolute path prefix when generating cue overlay, got %q", prefix)
	}
	err := fs.WalkDir(vfs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		f, err := vfs.Open(path)
		if err != nil {
			return err
		}
		defer f.Close() // nolint: errcheck

		b, err := io.ReadAll(f)
		if err != nil {
			return err
		}

		overlay[filepath.Join(prefix, path)] = load.FromBytes(b)
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// RandSeq produces random (basic, not crypto) letters of a given length.
func RandSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// ToInstanceDef exports the provided cue.Value (which is expected to be a thema
// schema) into a top-level definition within a cue.Instance, allowing some
// older cue stdlib (like encoding/openapi) that still want a cue.Instance to
// work with the value.
func ToInstanceDef(v cue.Value, name string, ctx *cue.Context) (*cue.Instance, error) {
	if ctx == nil {
		ctx = v.Context()
	}

	expr, _ := parser.ParseExpr("empty", "{}")
	bv := ctx.BuildExpr(expr)

	base := RandSeq(10)
	p := cue.MakePath(cue.Str(base), cue.Def(name))
	pp := cue.MakePath(cue.Str(base))
	dumpv := bv.FillPath(p, v).LookupPath(pp)

	// TODO just Eval()'ing without giving the user choices is not great.
	// But how else to get rid of all the (potential) joinSchema references?
	n := tastutil.Format(dumpv.Eval())
	var f *ast.File
	switch x := n.(type) {
	case *ast.StructLit:
		// errs not possible here, structlits always convert
		f, _ = astutil.ToFile(x) // nolint: errcheck
	case *ast.File:
		f = x
	default:
		return nil, fmt.Errorf("schema cue.Value converted to unexpected ast type %T", n)
	}

	// Quote labels that would shadow keywords/type identifiers. This is probably
	// something that cue's format package should do itself.
	ast.Walk(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Field:
			nam, isident, err := ast.LabelName(x.Label)
			if err == nil && isident && mustQuote(nam) {
				x.Label = ast.NewString(nam)
			}
		}
		return true

	}, nil)

	rt := (*cue.Runtime)(ctx)
	return rt.CompileFile(f)
}

// TODO make a better list - rely on CUE Go API somehow? Tokens?
func mustQuote(n string) bool {
	quoteneed := []string{
		"string",
		"number",
		"int",
		"uint",
		"float",
		"byte",
	}

	for _, s := range quoteneed {
		if n == s {
			return true
		}
	}
	return false
}
