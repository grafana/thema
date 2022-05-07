package util

import (
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"path/filepath"

	"cuelang.org/go/cue/load"
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

// AsInstance goes the long, awkward way around deprecation of cue.Instance to
// produce one. Export a cue.Value, then re-load and build it, and then use the
// Value.Context() of the input to create an Instance.
//
// This may not work with external references, since we lack the context to
// re-build them. Yuck. Cool, though, is that Value.Syntax() does at least generate
// the import statements...
// func AsInstance(v, space cue.Value, name cue.Selector) (*cue.Instance, error) {
// 	base := RandSeq(10)
// 	p := cue.MakePath(cue.Str(base), name)
// 	pp := cue.MakePath(cue.Str(base))
// 	dumpv := space.FillPath(p, v).LookupPath(pp)
//
// 	syn := dumpv.Syntax(
// 		cue.Definitions(true),
// 		cue.Hidden(true),
// 		cue.Optional(true),
// 		cue.Attributes(true),
// 		cue.Docs(true),
// 	)
//
// ast.Walk(syn, func(n ast.Node) bool {
// 	fmt.Printf("%T %+v\n", n, n)
// 	return true
// }, func(n ast.Node) {})
//
// return format.Node(syn,
// format.TabIndent(true),
// format.Simplify(),
// )
//
// 	// Normalize to ast.File
// 	var f *ast.File
// 	switch syns := syn.(type) {
// 	case *ast.StructLit:
// 		f.Decls = syns.Elts
// 	case *ast.File:
// 	default:
// 		return nil, fmt.Errorf("schema cue.Value converted to unexpected ast type %T", syn)
// 	}
//
// 	ctx := (*cue.Runtime)(space.Context())
// 	inst, err := ctx.CompileFile(f)
//
// }
