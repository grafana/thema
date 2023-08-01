package gocode

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"github.com/dave/dst/decorator"
	"github.com/dave/dst/dstutil"
	"golang.org/x/tools/imports"

	"github.com/grafana/thema/internal/util"
)

type GenGoFile struct {
	ErrIfAdd bool
	Path     string
	Appliers []dstutil.ApplyFunc
	In       []byte
}

func PostprocessGoFile(cfg GenGoFile) ([]byte, error) {
	fname := util.SanitizeLabelString(filepath.Base(cfg.Path))
	buf := new(bytes.Buffer)
	fset := token.NewFileSet()
	gf, err := decorator.ParseFile(fset, fname, string(cfg.In), parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("error parsing generated file: %w", err)
	}

	for _, af := range cfg.Appliers {
		dstutil.Apply(gf, af, nil)
	}

	err = decorator.Fprint(buf, gf)
	if err != nil {
		return nil, fmt.Errorf("error formatting generated file: %w", err)
	}

	byt, err := imports.Process(fname, buf.Bytes(), nil)
	if err != nil {
		return nil, fmt.Errorf("goimports processing of generated file failed: %w", err)
	}

	if cfg.ErrIfAdd {
		// Compare imports before and after; warn about performance if some were added
		gfa, _ := parser.ParseFile(fset, fname, string(byt), parser.ParseComments)
		imap := make(map[string]bool)
		for _, im := range gf.Imports {
			imap[im.Path.Value] = true
		}
		var added []string
		for _, im := range gfa.Imports {
			if !imap[im.Path.Value] {
				added = append(added, im.Path.Value)
			}
		}

		if len(added) != 0 {
			// TODO improve the guidance in this error if/when we better abstract over imports to generate
			return nil, fmt.Errorf("goimports added the following import statements to %s: \n\t%s\nRelying on goimports to find imports significantly slows down code generation. Either add these imports with an AST manipulation in cfg.ApplyFuncs, or set cfg.IgnoreDiscoveredImports to true", cfg.Path, strings.Join(added, "\n\t"))
		}
	}
	return byt, nil
}
