package typescript

import (
	"embed"
	"fmt"
	"strings"
	"text/template"
	"time"

	"cuelang.org/go/cue"
	"github.com/grafana/cuetsy"
	"github.com/grafana/cuetsy/ts/ast"
	"github.com/grafana/thema"
)

// All the parsed templates in the tmpl subdirectory
var tmpls *template.Template

//go:embed *.tmpl
var tmplFS embed.FS

func init() {
	base := template.New("tsgen").Funcs(template.FuncMap{
		"now": time.Now,
	})
	tmpls = template.Must(base.ParseFS(tmplFS, "*.tmpl"))
}

// TypeConfig governs the behavior of [GenerateTypes].
type TypeConfig struct {
	// CuetsyConfig is passed directly to cuetsy, the underlying code generator.
	//
	// If nil provided, defaults to Export: true.
	CuetsyConfig *cuetsy.Config

	// Group indicates that the type is a grouped lineage - the root schema itself
	// does not represent an object that is ever expected to exist independently,
	// but each of its top-level fields do.
	//
	// NOTE - https://github.com/grafana/thema/issues/62 is the issue for formalizing
	// the group concept. Fixing that issue will obviate this field. Once fixed,
	// this field will be ignored, deprecated, and eventually removed.
	Group bool

	// RootName specifies the name to use for the type representing the root of the
	// schema. If empty, this defaults to titlecasing of the lineage name.
	//
	// No-op if Group is true.
	RootName string

	// RootAsType controls whether the root schema is generated as a TypeScript
	// interface type (false) or alias type (true).
	//
	// No-op if Group is true.
	RootAsType bool
}

// GenerateTypes generates native TypeScript types and defaults corresponding to
// the provided Schema.
func GenerateTypes(sch thema.Schema, cfg *TypeConfig) (*ast.File, error) {
	if cfg == nil {
		cfg = new(TypeConfig)
	}
	if cfg.CuetsyConfig == nil {
		cfg.CuetsyConfig = &cuetsy.Config{
			Export: true,
		}
	}
	if cfg.RootName == "" {
		cfg.RootName = strings.Title(sch.Lineage().Name())
	}

	schdef := sch.Underlying().LookupPath(cue.MakePath(cue.Hid("_#schema", "github.com/grafana/thema")))
	tf, err := cuetsy.GenerateAST(schdef, *cfg.CuetsyConfig)
	if err != nil {
		return nil, fmt.Errorf("generating TS for child elements of schema failed: %w", err)
	}

	if !cfg.Group {
		as := cuetsy.TypeInterface
		if cfg.RootAsType {
			as = cuetsy.TypeAlias
		}
		top, err := cuetsy.GenerateSingleAST(cfg.RootName, schdef, as)
		if err != nil {
			return nil, fmt.Errorf("generating TS for schema root failed: %w", err)
		}
		tf.Nodes = append(tf.Nodes, top.T)
		if top.D != nil {
			tf.Nodes = append(tf.Nodes, top.D)
		}
	}

	return tf, nil
}
