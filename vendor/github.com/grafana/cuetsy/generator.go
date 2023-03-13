package cuetsy

import (
	"bytes"
	"fmt"
	"math/bits"
	"sort"
	"strings"
	"text/template"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/errors"
	"github.com/grafana/cuetsy/ts"
	tsast "github.com/grafana/cuetsy/ts/ast"
)

const (
	attrname        = "cuetsy"
	attrEnumDefault = "enumDefault"
	attrEnumMembers = "memberNames"
	attrKind        = "kind"
	attrForceText   = "forceText"
)

// TSType strings indicate the kind of TypeScript declaration to which a CUE
// value should be translated. They are used in both @cuetsy attributes, and in
// calls to certain methods.
type TSType string

const (
	// TypeAlias targets conversion of a CUE value to a TypeScript `type`
	// declaration, which are called type aliases:
	// https://www.typescriptlang.org/docs/handbook/2/everyday-types.html#type-aliases
	TypeAlias TSType = "type"

	// TypeInterface targets conversion of a CUE value to a TypeScript `interface`
	// declaration:
	// https://www.typescriptlang.org/docs/handbook/2/everyday-types.html#interfaces
	TypeInterface TSType = "interface"

	// TypeEnum targets conversion of a CUE value to a TypeScript `enum`
	// declaration:
	// https://www.typescriptlang.org/docs/handbook/2/everyday-types.html#enums
	TypeEnum TSType = "enum"
)

var allKinds = [...]TSType{
	TypeAlias,
	TypeInterface,
	TypeEnum,
}

// An ImportMapper takes an ImportDecl and returns a string indicating the
// import statement that should be used in the corresponding typescript, or
// an error if no mapping can be made.
type ImportMapper func(*ast.ImportDecl) (string, error)

// NoImportMappingErr returns a standard error indicating that no mapping can be
// made for the provided import statement.
func NoImportMappingErr(d *ast.ImportDecl) error {
	return errors.Newf(d.Pos(), "a corresponding typescript import is not available for %q", d.Import.String())
}

func nilImportMapper(d *ast.ImportDecl) (string, error) { return "", NoImportMappingErr(d) }

// Config governs certain variable behaviors when converting CUE to Typescript.
type Config struct {
	// ImportMapper determines how CUE imports are mapped to Typescript imports.
	// If nil, any non-stdlib import in the CUE source will result in a fatal
	// error.
	ImportMapper

	// Export determines whether generated TypeScript symbols are exported.
	Export bool
}

// Generate takes a cue.Value and generates the corresponding TypeScript for all
// top-level members of that value that have appropriate @cuetsy attributes.
//
// Hidden fields are ignored.
func Generate(val cue.Value, c Config) (b []byte, err error) {
	file, err := GenerateAST(val, c)
	if err != nil {
		return nil, err
	}
	return []byte("\n" + file.String()), nil
}

func GenerateAST(val cue.Value, c Config) (*ts.File, error) {
	if err := val.Validate(); err != nil {
		return nil, err
	}

	if c.ImportMapper == nil {
		c.ImportMapper = nilImportMapper
	}

	g := &generator{
		c:   c,
		val: &val,
	}

	iter, err := val.Fields(
		cue.Definitions(true),
		cue.Optional(true),
	)
	if err != nil {
		return nil, err
	}

	var file ts.File
	for iter.Next() {
		n := g.decl(iter.Selector().String(), iter.Value())
		file.Nodes = append(file.Nodes, n...)
	}

	return &file, g.err
}

func GenerateSingleAST(name string, v cue.Value, t TSType) (*DeclPair, error) {
	g := &generator{
		c:   Config{Export: true},
		val: &v,
	}

	switch t {
	case TypeEnum:
		return fromDeclSlice(g.genEnum(name, v), g.err)
	case TypeInterface:
		return fromDeclSlice(g.genInterface(name, v), g.err)
	case TypeAlias:
		return fromDeclSlice(g.genType(name, v), g.err)
	default:
		return nil, fmt.Errorf("unrecognized TSType %q", string(t))
	}
}

// DeclPair represents a generated type declaration, with its corresponding default declaration.
type DeclPair struct {
	// The generated type declaration.
	T ts.Decl
	// The default declaration corresponding to T.
	D ts.Decl
}

func fromDeclSlice(decl []ts.Decl, err error) (*DeclPair, error) {
	if err != nil {
		return nil, err
	}
	switch len(decl) {
	case 0:
		return nil, errors.New("no decls returned")
	case 1:
		return &DeclPair{
			T: decl[0],
		}, nil
	case 2:
		return &DeclPair{
			T: decl[0],
			D: decl[1],
		}, nil
	default:
		return nil, fmt.Errorf("expected 1 or 2 decls in slice, got %v", len(decl))
	}
}

type generator struct {
	val *cue.Value
	c   Config
	err errors.Error
}

func (g *generator) addErr(err error) {
	if err != nil {
		g.err = errors.Append(g.err, errors.Promote(err, "generate failed"))
	}
}

func execGetString(t *template.Template, data interface{}) (string, error) {
	var tpl bytes.Buffer
	if err := t.Execute(&tpl, data); err != nil {
		return "", err
	}
	result := tpl.String()
	return result, nil
}

func (g *generator) decl(name string, v cue.Value) []ts.Decl {
	tst, err := getKindFor(v)
	if err != nil {
		// Ignore values without attributes
		return nil
	}
	switch tst {
	case TypeEnum:
		return g.genEnum(name, v)
	case TypeInterface:
		return g.genInterface(name, v)
	case TypeAlias:
		return g.genType(name, v)
	default:
		return nil // TODO error out
	}
}

func (g *generator) genType(name string, v cue.Value) []ts.Decl {
	var tokens []tsast.Expr
	// If there's an AndOp first, pass through it.
	op, dvals := v.Expr()
	if op == cue.AndOp {
		op, dvals = dvals[0].Expr()
	}
	switch op {
	case cue.OrOp:
		for _, dv := range dvals {
			tok, err := tsprintField(dv, true)
			if err != nil {
				g.addErr(err)
				return nil
			}
			tokens = append(tokens, tok)
		}
	case cue.NoOp, cue.RegexMatchOp:
		tok, err := tsprintField(v, true)
		if err != nil {
			g.addErr(err)
			return nil
		}
		tokens = append(tokens, tok)
	default:
		g.addErr(valError(v, "typescript types may only be generated from a single value or disjunction of values"))
	}

	ret := make([]ts.Decl, 2)

	ret[0] = tsast.TypeDecl{
		Name:        ts.Ident(name),
		Type:        tsast.BasicType{Expr: ts.Union(tokens...)},
		CommentList: commentsFor(v, true),
		Export:      g.c.Export,
	}

	d, ok := v.Default()
	if !ok {
		return ret[:1]
	}

	val, err := tsprintField(d, false)
	g.addErr(err)

	def := tsast.VarDecl{
		Names:  ts.Names("default" + name),
		Type:   ts.Ident(name),
		Value:  val,
		Export: g.c.Export,
	}

	// Only make struct-kinded types into partials
	if v.IncompleteKind() == cue.StructKind {
		def.Type = tsast.TypeTransformExpr{
			Transform: "Partial",
			Expr:      def.Type,
		}
	}

	ret[1] = def
	return ret
}

type KV struct {
	K, V string
}

// genEnum turns the following cue values into typescript enums:
//   - value disjunction (a | b | c): values are taken as attribute memberNames,
//     if memberNames is absent, then keys implicitly generated as CamelCase
//   - string struct: struct keys get enum keys, struct values enum values
func (g *generator) genEnum(name string, v cue.Value) []ts.Decl {
	vdoc := v.Doc()
	// FIXME compensate for attribute-applying call to Unify() on incoming Value
	op, dvals := v.Expr()
	if op == cue.AndOp {
		v = dvals[0]
		op, _ = v.Expr()
	}

	// We restrict the expression of TS enums to ints or strings.
	allowed := cue.StringKind | cue.IntKind
	ik := v.IncompleteKind()
	if ik&allowed != ik {
		g.addErr(valError(v, "typescript enums may only be generated from concrete strings, or ints with memberNames attribute"))
		return nil
	}

	exprs, err := orEnum(v)
	if err != nil {
		g.addErr(err)
	}

	ret := make([]ts.Decl, 2)
	ret[0] = tsast.TypeDecl{
		Name:        ts.Ident(name),
		Type:        tsast.EnumType{Elems: exprs},
		CommentList: commentsForGroup(vdoc, true),
		Export:      g.c.Export,
	}

	defaultIdent, err := enumDefault(v)
	g.addErr(err)

	if defaultIdent == nil {
		return ret[:1]
	}

	ret[1] = tsast.VarDecl{
		Names:  ts.Names("default" + name),
		Type:   ts.Ident(name),
		Value:  tsast.SelectorExpr{Expr: ts.Ident(name), Sel: *defaultIdent},
		Export: g.c.Export,
	}
	return ret
}

func enumDefault(v cue.Value) (*tsast.Ident, error) {
	def, ok := v.Default()
	if !ok {
		return nil, def.Err()
	}

	if v.IncompleteKind() == cue.StringKind {
		s, _ := def.String()
		return &tsast.Ident{Name: strings.Title(s)}, nil
	}

	// For Int, Float, Numeric we need to find the default value and its corresponding memberName value
	a := v.Attribute(attrname)
	val, found, err := a.Lookup(0, attrEnumMembers)
	if err != nil || !found {
		panic(fmt.Sprintf("looking up memberNames: found=%t err=%s", found, err))
	}
	evals := strings.Split(val, "|")

	_, dvals := v.Expr()
	for i, val := range dvals {
		valLab, _ := val.Label()
		defLab, _ := def.Label()
		if valLab == defLab {
			return &tsast.Ident{Name: evals[i]}, nil
		}
	}

	// should never reach here tho
	return nil, valError(v, "unable to find memberName corresponding to the default")
}

// List the pairs of values and member names in an enum. Err if input is not an enum
func enumPairs(v cue.Value) ([]enumPair, error) {
	// TODO should validate here. Or really, this is just evidence of how building these needs its own types
	op, dvals := v.Expr()
	if !targetsKind(v, TypeEnum) || op != cue.OrOp {
		return nil, fmt.Errorf("not an enum: %v (%s)", v, v.Path())
	}

	a := v.Attribute(attrname)
	val, found, err := a.Lookup(0, attrEnumMembers)
	if err != nil {
		panic(fmt.Sprintf("looking up memberNames: found=%t err=%s", found, err))
	}

	var evals []string
	if found {
		evals = strings.Split(val, "|")
	} else if v.IncompleteKind() == cue.StringKind {
		for _, part := range dvals {
			s, _ := part.String()
			evals = append(evals, strings.Title(s))
		}
	} else {
		return nil, fmt.Errorf("must provide memberNames attribute for non-string enums")
	}

	var pairs []enumPair
	for i, eval := range evals {
		pairs = append(pairs, enumPair{
			name: eval,
			val:  dvals[i],
		})
	}

	return pairs, nil
}

type enumPair struct {
	name string
	val  cue.Value
}

func orEnum(v cue.Value) ([]ts.Expr, error) {
	_, dvals := v.Expr()
	a := v.Attribute(attrname)

	var attrMemberNameExist bool
	var evals []string
	if a.Err() == nil {
		val, found, err := a.Lookup(0, attrEnumMembers)
		if err == nil && found {
			attrMemberNameExist = true
			evals = strings.Split(val, "|")
			if len(evals) != len(dvals) {
				return nil, valError(v, "typescript enums and %s attributes size doesn't match", attrEnumMembers)
			}
		}
	}

	// We only allowed String Enum to be generated without memberName attribute
	if v.IncompleteKind() != cue.StringKind && !attrMemberNameExist {
		return nil, valError(v, "typescript numeric enums may only be generated from memberNames attribute")
	}

	var fields []ts.Expr
	for idx, dv := range dvals {
		var text string
		var id tsast.Ident
		if attrMemberNameExist {
			text = evals[idx]
			id = ts.Ident(text)
		} else {
			text, _ = dv.String()
			id = ts.Ident(strings.Title(text))
		}

		if !dv.IsConcrete() {
			return nil, valError(v, "typescript enums may only be generated from a disjunction of concrete strings or numbers")
		}

		if id.Validate() != nil {
			return nil, valError(v, "title casing of enum member %q produces an invalid typescript identifier; memberNames must be explicitly given in @cuetsy attribute", text)
		}

		fields = append(fields, tsast.AssignExpr{
			// Simple mapping of all enum values (which we are assuming are in
			// lowerCamelCase) to corresponding CamelCase
			Name:  id,
			Value: tsprintConcrete(dv),
		})
	}

	sort.Slice(fields, func(i, j int) bool {
		return fields[i].String() < fields[j].String()
	})

	return fields, nil
}

func (g *generator) genInterface(name string, v cue.Value) []ts.Decl {
	// We restrict the derivation of Typescript interfaces to struct kinds.
	// (More than just a struct literal match this, though.)
	if v.IncompleteKind() != cue.StructKind {
		// FIXME check for bottom here, give different error
		g.addErr(valError(v, "typescript interfaces may only be generated from structs"))
		return nil
	}

	extends, nolit, err := findExtends(v)
	if err != nil {
		g.addErr(err)
		return nil
	}
	var elems []tsast.KeyValueExpr
	var defs []tsast.KeyValueExpr

	iter, _ := v.Fields(cue.Optional(true))
	for iter != nil && iter.Next() {
		if iter.Selector().PkgPath() != "" {
			g.addErr(valError(iter.Value(), "cannot generate hidden fields; typescript has no corresponding concept"))
			return nil
		}

		// Skip fields that are subsumed by the Value representing the
		// unification of all refs that will be represented using an "extends"
		// keyword.
		//
		// This does introduce the possibility that even some fields which are
		// literally declared on the struct will not end up written out in
		// Typescript (though the semantics will still be correct). That's
		// likely to be a bit confusing for users, but we have no choice. The
		// (preferable) alternative would rely on Unify() calls to build a Value
		// containing only those fields that we want, then iterating over that
		// in this loop.
		//
		// Unfortunately, as of v0.4.0, Unify() appears to not preserve
		// attributes on the Values it generates, which makes it impossible to
		// rely on, as the tsprintField() func later also needs to check these
		// attributes in order to decide whether to render a field as a
		// reference or a literal.
		//
		// There's _probably_ a way around this, especially when we move to an
		// AST rather than dumb string templates. But i'm tired of looking.
		if len(extends) > 0 {
			// Look up the path of the current field within the nolit value,
			// then check it for subsumption.
			sel := iter.Selector()
			if iter.IsOptional() {
				sel = sel.Optional()
			}

			sub := nolit.LookupPath(cue.MakePath(sel))
			// Theoretically, lattice equality can be defined as bijective
			// subsumption. In practice, Subsume() seems to ignore optional
			// fields, and Equals() doesn't. So, use Equals().
			if sub.Exists() && sub.Equals(iter.Value()) {
				continue
			}
		}

		k := iter.Selector().String()
		if iter.IsOptional() {
			k += "?"
		}

		tref, err := g.genInterfaceField(iter.Value())
		if err != nil || tref == nil {
			return nil
		}

		elems = append(elems, tsast.KeyValueExpr{
			Key:         ts.Ident(k),
			Value:       tref.T,
			CommentList: commentsFor(iter.Value(), true),
		})

		if tref.D != nil {
			defs = append(defs, tsast.KeyValueExpr{
				Key:   ts.Ident(strings.TrimSuffix(k, "?")),
				Value: tref.D,
			})
		}
	}

	sort.Slice(elems, func(i, j int) bool {
		return elems[i].Key.String() < elems[j].Key.String()
	})
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].Key.String() < defs[j].Key.String()
	})

	ret := make([]ts.Decl, 2)

	ret[0] = tsast.TypeDecl{
		Name: ts.Ident(name),
		Type: tsast.InterfaceType{
			Elems:   elems,
			Extends: extends,
		},
		CommentList: commentsFor(v, true),
		Export:      g.c.Export,
	}

	if len(defs) == 0 {
		return ret[:1]
	}

	ret[1] = tsast.VarDecl{
		Names: ts.Names("default" + name),
		Type: tsast.TypeTransformExpr{
			Transform: "Partial",
			Expr:      ts.Ident(name),
		},
		Value:  tsast.ObjectLit{Elems: defs},
		Export: g.c.Export,
	}

	return ret
}

// Recursively walk down Values returned from Expr() and separate
// unified/embedded structs from a struct literal, so that we can make the
// former (if they are also marked with @cuetsy(kind="interface")) show up
// as "extends" instead of inlining their fields.
func findExtends(v cue.Value) ([]ts.Expr, cue.Value, error) {
	var extends []ts.Expr
	// Create an empty value, onto which we'll unify fields that need not be
	// generated as literals.
	baseNolit := v.Context().CompileString("")
	nolit := v.Context().CompileString("")
	var walkExpr func(v cue.Value) error
	walkExpr = func(v cue.Value) error {
		op, dvals := v.Expr()
		switch op {
		case cue.NoOp:
			// Simple path - when the field is a plain struct literal decl, the walk function
			// will take this branch and return immediately.

			// FIXME this does the struct literal path correctly, but it also
			// catches this case, for some reason:
			//
			//   Thing: {
			//       other.Thing
			//   }
			//
			// The saner form - `Thing: other.Thing` - does not go through this path.
			return nil
		case cue.OrOp:
			return valError(v, "typescript interfaces cannot be constructed from disjunctions")
		case cue.SelectorOp:
			expr, err := refAsInterface(v)
			if err != nil {
				return err
			}

			// If we have a string to add to the list of "extends", then also
			// add the ref to the list of fields to exclude if subsumed.
			if expr != nil {
				extends = append(extends, expr)
				nolit = baseNolit.Unify(nolit.Unify(cue.Dereference(v)))
			}
			return nil
		case cue.AndOp:
			// First, search the dvals for StructLits. Having more than one is possible,
			// but weird, as writing >1 literal and unifying them is the same as just writing
			// one containing the unified result - more complicated with no obvious benefit.
			for _, dv := range dvals {
				if dv.IncompleteKind() != cue.StructKind && dv.IncompleteKind() != cue.TopKind {
					panic("impossible? seems like it should be. if this pops, clearly not!")
				}

				if err := walkExpr(dv); err != nil {
					return err
				}
			}
			return nil
		default:
			panic(fmt.Sprintf("unhandled op type %s", op.String()))
		}
	}

	if err := walkExpr(v); err != nil {
		return nil, nolit, err
	}

	return extends, nolit, nil
}

// Generate a typeRef for the cue.Value
func (g *generator) genInterfaceField(v cue.Value) (*typeRef, error) {
	tref := &typeRef{}
	var err error

	// Check if we've got an enum reference at top depth or one down. If we do, it
	// changes how we generate.
	if containsPred(v, 1,
		isReference,
		func(v cue.Value) bool { return targetsKind(cue.Dereference(v), TypeEnum) },
	) {
		return g.genEnumReference(v)
	}

	// One path for when there's a ref to a cuetsy node, and a separate one otherwise
	if !containsCuetsyReference(v) {
		tref.T, err = tsprintField(v, true)
		if err != nil {
			g.addErr(valError(v, "could not generate field: %w", err))
			return nil, err
		}
	} else {
		expr, err := tsprintField(v, true)
		if err != nil {
			g.addErr(err)
			return nil, nil
		}
		tref.T = expr

		// Deconstruct the field's expressions.
		// conjuncts := appendSplit(nil, cue.AndOp, v)

		// var expr ts.Expr
		//
		// for _, cv := range conjuncts {
		// 	disjuncts := appendSplit(nil, cue.OrOp, cv)
		// 	for i, dv := range disjuncts {
		// 		if _, r := dv.Reference(); len(r) == 0 {
		// 			disjuncts[i] = dv.Eval()
		// 		}
		// 	}
		// 	switch len(disjuncts) {
		// 	case 0:
		// 		// conjunct eliminated - need more preprocessing to actually make this possible
		// 		panic("TODO, unreachable")
		// 	case 1:
		// 		err := disjuncts[0].Err()
		// 		if err != nil {
		// 			g.addErr(valError(v, "invalid value"))
		// 			return nil
		// 		}
		// 		expr, err = tsprintField(disjuncts[0])
		// 		if err != nil {
		// 			g.addErr(valError(v, "invalid value"))
		// 			return nil
		// 		}
		// 	default:
		// 		// TODO create disjunction handler
		// 	}
		// }
	}

	exists, defExpr, err := tsPrintDefault(v)
	if exists {
		tref.D = defExpr
	}
	g.addErr(err)
	return tref, err
}

// Generate a typeref for a value that refers to a field
func (g *generator) genEnumReference(v cue.Value) (*typeRef, error) {
	var lit *cue.Value

	findIdent := func(ev, tv cue.Value) (*tsast.Ident, error) {
		if ev.Subsume(tv) != nil {
			err := valError(v, "may only apply values to an enum that are members of that enum; %#v is not a member of %#v", tv, ev)
			g.addErr(err)
			return nil, err
		}
		pairs, err := enumPairs(ev)
		if err != nil {
			return nil, err
		}
		for _, pair := range pairs {
			if veq(pair.val, tv) {
				return &tsast.Ident{Name: pair.name}, nil
			}
		}

		panic(fmt.Sprintf("unreachable - %#v not equal to any member of %#v, but should have been caught by subsume check", tv, ev))
	}

	conjuncts := appendSplit(nil, cue.AndOp, v)
	switch len(conjuncts) {
	case 0:
		panic("unreachable")
	case 1:
	case 2:
		// The only case we actually want to support, at least for now, is this:
		//
		//   enum: "foo" | "bar" @cuetsy(kind="enum")
		//   enumref: enum & "foo" @cuetsy(kind="type")
		//
		// Where we render enumref to TS as `Enumref: Enum.Foo`.
		// For that case, we allow at most two conjuncts, and make sure they
		// fit the pattern of the two operands above.
		aref, bref := isReference(conjuncts[0]), isReference(conjuncts[1])
		aconc, bconc := conjuncts[0].IsConcrete(), conjuncts[1].IsConcrete()
		var cr cue.Value
		if aref {
			cr, lit = conjuncts[0], &(conjuncts[1])
		} else {
			cr, lit = conjuncts[1], &(conjuncts[0])
		}
		if aref == bref || aconc == bconc || cr.Subsume(*lit) != nil {
			ve := valError(v, "may only unify a referenced enum with a concrete literal member of that enum")
			g.addErr(ve)
			return nil, ve
		}

	default:
		ve := valError(v, "complex unifications containing references to enums are not currently supported")
		g.addErr(ve)
		return nil, ve
	}

	// Search the expr tree for the actual enum. This approach is uncomfortable
	// without having the assurance that there aren't more than one possible match/a
	// guarantee from the CUE API of a stable, deterministic search order, etc.
	ev, referrer, has := findRefWithKind(v, TypeEnum)
	if !has {
		ve := valError(v, "does not reference a field with a cuetsy enum attribute")
		g.addErr(ve)
		return nil, fmt.Errorf("no enum attr in %s", v)
	}

	var err error
	decls := g.genEnum("foo", ev)
	ref := &typeRef{}

	// Construct the type component of the reference
	switch len(decls) {
	default:
		ve := valError(v, "unsupported number of expression args (%v) in reference, expected 1 or 2", len(decls))
		g.addErr(ve)
		return nil, ve
	case 1, 2:
		ref.T, err = referenceValueAs(referrer)
		if err != nil {
			panic(err)
		}
	}

	// Either specify a default if one exists (one conjunct), or rewrite the type to
	// reference one of the members of the enum (two conjuncts).
	switch len(conjuncts) {
	case 1:
		if defv, hasdef := v.Default(); hasdef {
			if defaultIdent, err := findIdent(ev, defv); err == nil {
				ref.D = tsast.SelectorExpr{Expr: ref.T, Sel: *defaultIdent}
			} else {
				return nil, err
			}
		}
	case 2:
		if typeIdent, err := findIdent(ev, *lit); err == nil {
			ref.T = tsast.SelectorExpr{
				Expr: ref.T,
				Sel:  *typeIdent,
			}
		} else {
			return nil, err
		}
	}

	return ref, nil
}

// typeRef is a pair of expressions for referring to another type - the reference
// to the type, and the default value for the referrer. The default value
// may be the one provided by either the referent, or by the field doing the referring
// (in the case of a superseding mark).
type typeRef struct {
	T ts.Expr
	D ts.Expr
}

func tsPrintDefault(v cue.Value) (bool, ts.Expr, error) {
	d, ok := v.Default()
	// [...number] results in [], which is a fake default, we need to correct it here.
	// if ok && d.Kind() == cue.ListKind {
	// 	len, err := d.Len().Int64()
	// 	if err != nil {
	// 		return false, nil, err
	// 	}
	// 	var defaultExist bool
	// 	if len <= 0 {
	// 		op, vals := v.Expr()
	// 		if op == cue.OrOp {
	// 			for _, val := range vals {
	// 				vallen, _ := d.Len().Int64()
	// 				if val.Kind() == cue.ListKind && vallen <= 0 {
	// 					defaultExist = true
	// 					break
	// 				}
	// 			}
	// 			if !defaultExist {
	// 				ok = false
	// 			}
	// 		} else {
	// 			ok = false
	// 		}
	// 	}
	// }

	if ok {
		expr, err := tsprintField(d, false)
		if err != nil {
			return false, nil, err
		}

		if isReference(d) {
			switch t := expr.(type) {
			case tsast.SelectorExpr:
				t.Sel.Name = "default" + t.Sel.Name
				expr = t
			case tsast.Ident:
				t.Name = "default" + t.Name
				expr = t
			default:
				panic(fmt.Sprintf("unexpected type %T", expr))
			}
		}

		return true, expr, nil
	}
	return false, nil, nil
}

// Render a string containing a Typescript semantic equivalent to the provided
// Value for placement in a single field, if possible.
func tsprintField(v cue.Value, isType bool) (ts.Expr, error) {
	// Let the forceText attribute supersede everything.
	if ft := getForceText(v); ft != "" {
		return ts.Raw(ft), nil
	}

	// References are orthogonal to the Kind system. Handle them first.
	if containsCuetsyReference(v) {
		ref, err := referenceValueAs(v)
		if err != nil {
			return nil, err
		}
		if ref != nil {
			return ref, nil
		}
		return nil, valError(v, "failed to generate reference correctly for path %s", v.Path().String())
	}

	verr := v.Validate(cue.Final())
	if verr != nil {
		return nil, verr
	}

	op, dvals := v.Expr()
	// Eliminate concretes first, to make handling the others easier.

	// Concrete values.
	// Includes "foobar", 5, [1,2,3], etc. (literal values)
	k := v.Kind()
	switch k {
	case cue.StructKind:
		switch op {
		case cue.SelectorOp, cue.AndOp, cue.NoOp:
			// Checks [string]something only.
			// It skips structs like {...} (cue.TopKind) to avoid undesired results.
			val := v.LookupPath(cue.MakePath(cue.AnyString))
			if val.Exists() && val.IncompleteKind() != cue.TopKind {
				expr, err := tsprintField(val, isType)
				if err != nil {
					return nil, valError(v, err.Error())
				}
				kvs := []tsast.KeyValueExpr{
					{
						Key:         ts.Ident("string"),
						Value:       expr,
						CommentList: commentsFor(val.Value(), true),
					},
				}
				return tsast.ObjectLit{Elems: kvs, IsType: isType, IsMap: true}, nil
			}

			iter, err := v.Fields(cue.Optional(true))
			if err != nil {
				return nil, valError(v, "something went wrong when generate nested structs")
			}
			size, _ := v.Len().Int64()
			kvs := make([]tsast.KeyValueExpr, 0, size)
			for iter.Next() {
				expr, err := tsprintField(iter.Value(), isType)
				if err != nil {
					return nil, valError(v, err.Error())
				}
				k := iter.Label()
				if iter.IsOptional() {
					k += "?"
				}
				kvs = append(kvs, tsast.KeyValueExpr{
					Key:         ts.Ident(k),
					Value:       expr,
					CommentList: commentsFor(iter.Value(), true),
				})
			}

			return tsast.ObjectLit{Elems: kvs, IsType: isType}, nil
		default:
			panic(fmt.Sprintf("not expecting op type %d", op))
		}
	case cue.ListKind:
		// A list is concrete (and thus its complete kind is ListKind instead of
		// BottomKind) iff it specifies a finite number of elements - is
		// "closed". This is independent of the types of its elements, which may
		// be anywhere on the concreteness spectrum.
		//
		// For closed lists, we simply iterate over its component elements and
		// print their typescript representation.
		iter, _ := v.List()
		var elems []ts.Expr
		for iter.Next() {
			e, err := tsprintField(iter.Value(), isType)
			if err != nil {
				return nil, err
			}
			elems = append(elems, e)
		}
		return ts.List(elems...), nil
	case cue.StringKind, cue.BoolKind, cue.FloatKind, cue.IntKind:
		return tsprintConcrete(v), nil
	case cue.BytesKind:
		return nil, valError(v, "bytes have no equivalent in Typescript; use double-quotes (string) instead")
	}

	// Handler for disjunctions
	disj := func(dvals []cue.Value) (ts.Expr, error) {
		parts := make([]ts.Expr, 0, len(dvals))
		for _, dv := range dvals {
			p, err := tsprintField(dv, isType)
			if err != nil {
				return nil, err
			}
			parts = append(parts, p)
		}
		return ts.Union(parts...), nil
	}

	// Others: disjunctions, etc.
	ik := v.IncompleteKind()
	switch ik {
	case cue.BottomKind:
		return nil, valError(v, "bottom, unsatisfiable")
	case cue.ListKind:
		// This list is open - its final element is ...<value> - and we can only
		// meaningfully convert open lists to typescript if there are zero other
		// elements.

		// First, peel off a simple default, if one exists.
		// dlist, has := v.Default()
		// if has && op == cue.OrOp {
		// 	di := analyzeList(dlist)
		// 	if len(dvals) != 2 {
		// 		panic(fmt.Sprintf("%v branches on list disjunct, can only handle 2", len(dvals)))
		// 	}
		// 	if di.eq(analyzeList(dvals[1])) {
		// 		v = dvals[0]
		// 	} else if di.eq(analyzeList(dvals[0])) {
		// 		v = dvals[1]
		// 	} else {
		// 		panic("wat - list kind had default but analysis did not match for either disjunct branch")
		// 	}
		// }

		// If the default (all lists have a default, usually self, ugh) differs from the
		// input list, peel it off. Otherwise our AnyIndex lookup may end up getting
		// sent on the wrong path.
		defv, _ := v.Default()
		if !defv.Equals(v) {
			v = dvals[0]
		}

		e := v.LookupPath(cue.MakePath(cue.AnyIndex))
		if e.Exists() {
			expr, err := tsprintField(e, isType)
			if err != nil {
				return nil, err
			}
			return tsast.ListExpr{Expr: expr}, nil
		} else {
			panic("unreachable - open list must have a type")
		}
	case cue.NumberKind, cue.StringKind:
		// It appears there are only three cases in which we can have an
		// incomplete NumberKind or StringKind:
		//
		// 1. The corresponding literal is a bounding constraint (which subsumes
		// both int and float), e.g. >2.2, <"foo"
		// 2. There's a disjunction of concrete literals of the relevant type
		// 2. The corresponding literal is the basic type "number" or "string"
		//
		// The first case has no equivalent in typescript, and entails we error
		// out. The other two have the same handling as for other kinds, so we
		// fall through. We disambiguate by seeing if there is an expression
		// (other than Or, "|"), which is how ">" and "2.2" are represented.
		//
		// TODO get more certainty/a clearer way of ascertaining this
		switch op {
		case cue.RegexMatchOp:
			// Typescript has no native type for representing regexes. Only
			// option is to fall back to string.
			return tsprintType(cue.StringKind), nil
		case cue.NoOp, cue.OrOp, cue.AndOp:
		default:
			return nil, valError(v, "bounds constraints are not supported as they lack a direct typescript equivalent")
		}
		fallthrough
	case cue.FloatKind, cue.IntKind, cue.BoolKind, cue.NullKind, cue.StructKind:
		// Having eliminated the possibility of bounds/constraints, we're left
		// with disjunctions and basic types.
		switch op {
		case cue.OrOp:
			return disj(dvals)
		case cue.NoOp, cue.AndOp:
			// There's no op for simple unification; it's a basic type, and can
			// be trivially rendered.
		default:
			panic("unreachable...?")
		}
		fallthrough
	case cue.TopKind:
		return tsprintType(ik), nil
	case cue.BytesKind:
		return tsprintType(cue.StringKind), nil
	}

	// Having more than one possible kind entails a disjunction, TopKind, or
	// NumberKind. We've already eliminated TopKind and NumberKind, so now check
	// if there's more than one bit set. (If there isn't, it's a bug: we've
	// missed a kind above). If so, run our disjunction-handling logic.
	if bits.OnesCount16(uint16(ik)) > 1 {
		return disj(dvals)
	}

	return nil, valError(v, "unrecognized kind %v", ik)
}

// ONLY call this function if it has been established that the provided Value is
// Concrete.
func tsprintConcrete(v cue.Value) ts.Expr {
	switch v.Kind() {
	case cue.NullKind:
		return ts.Null()
	case cue.StringKind:
		s, _ := v.String()
		return ts.Str(s)
	case cue.FloatKind:
		f, _ := v.Float64()
		return ts.Float(f)
	case cue.NumberKind, cue.IntKind:
		i, _ := v.Int64()
		return ts.Int(i)
	case cue.BoolKind:
		b, _ := v.Bool()
		return ts.Bool(b)
	default:
		panic("unreachable")
	}
}

func tsprintType(k cue.Kind) ts.Expr {
	switch k {
	case cue.BoolKind:
		return ts.Ident("boolean")
	case cue.StringKind:
		return ts.Ident("string")
	case cue.NumberKind, cue.FloatKind, cue.IntKind:
		return ts.Ident("number")
	case cue.TopKind:
		return ts.Ident("unknown")
	default:
		return nil
	}
}

func valError(v cue.Value, format string, args ...interface{}) error {
	s := v.Source()
	if s == nil {
		return fmt.Errorf(format, args...)
	}
	return errors.Newf(s.Pos(), format, args...)
}

func refAsInterface(v cue.Value) (ts.Expr, error) {
	// Bail out right away if the value isn't a reference
	op, dvals := v.Expr()
	if !isReference(v) || op != cue.SelectorOp {
		return nil, fmt.Errorf("not a reference")
	}

	// Have to do attribute checks on the referenced field itself, so deref
	deref := cue.Dereference(v)
	dstr, _ := dvals[1].String()

	// FIXME It's horrifying, teasing out the type of selector kinds this way. *Horrifying*.
	switch dvals[0].Source().(type) {
	case nil:
		// A nil subject means an unqualified selector (no "."
		// literal).  This can only possibly be a reference to some
		// sibling or parent of the top-level Value being generated.
		// (We can't do cycle detection with the meager tools
		// exported in cuelang.org/go/cue, so all we have for the
		// parent case is hopium.)
		if _, ok := dvals[1].Source().(*ast.Ident); ok && targetsKind(deref, TypeInterface) {
			return ts.Ident(dstr), nil
		}
	case *ast.SelectorExpr:
		// panic("case 2")
		if targetsKind(deref, TypeInterface) {
			return ts.Ident(dstr), nil
		}
	case *ast.Ident:
		// panic("case 3")
		if targetsKind(deref, TypeInterface) {
			str, ok := dvals[0].Source().(fmt.Stringer)
			if !ok {
				panic("expected dvals[0].Source() to implement String()")
			}

			return tsast.SelectorExpr{
				Expr: ts.Ident(str.String()),
				Sel:  ts.Ident(dstr),
			}, nil
		}
	default:
		return nil, valError(v, "unknown selector subject type %T, cannot translate", dvals[0].Source())
	}

	return nil, nil
}

// referenceValueAs returns the string that should be used to create a Typescript
// reference to the given struct, if a reference is allowable.
//
// References are only permitted to other Values with an @cuetsy(kind)
// attribute. The variadic parameter determines which kinds will be treated as
// permissible. By default, all kinds are permitted.
//
// An nil expr indicates a reference is not allowable, including the case
// that the provided Value is not actually a reference. A non-nil error
// indicates a deeper problem.
func referenceValueAs(v cue.Value, kinds ...TSType) (ts.Expr, error) {
	// Bail out right away if there's no reference anywhere in the value.
	// if !containsReference(v) {
	// 	return nil, nil
	// }
	// End goal: we want to render a reference appropriately in Typescript.
	// If the top-level is a reference, then this is simple.
	//
	// If the top-level merely contains a reference, this is harder.
	// - Let's start by only supporting that case when it's because there's a default.

	// Calling Expr peels off all default paths.
	op, dvals := v.Expr()
	_ = op

	if !isReference(v) {
		_, has := v.Default()
		if hasOverrideValues(v) {
			v = dvals[1]
		} else if !has || !isReference(dvals[0]) {
			return nil, valError(v, "references within complex logic are currently unsupported")
		} else {
			v = dvals[0]
		}

		// This may break a bunch of things but let's see if it gives us a
		// defensible baseline
		op, dvals = v.Expr()
	}

	var dstr string
	if len(dvals) > 1 {
		dstr, _ = dvals[1].String()
	}

	// Have to do attribute checks on the referenced field itself, so deref
	deref := cue.Dereference(v)

	// FIXME It's horrifying, teasing out the type of selector kinds this way. *Horrifying*.
	switch dvals[0].Source().(type) {
	case nil:
		// A nil subject means an unqualified selector (no "."
		// literal).  This can only possibly be a reference to some
		// sibling or parent of the top-level Value being generated.
		// (We can't do cycle detection with the meager tools
		// exported in cuelang.org/go/cue, so all we have for the
		// parent case is hopium.)
		if _, ok := dvals[1].Source().(*ast.Ident); ok && targetsKind(deref, kinds...) {
			return ts.Ident(dstr), nil
		}
	case *ast.SelectorExpr:
		// panic("case 2")
		if targetsKind(deref, kinds...) {
			return ts.Ident(dstr), nil
		}
	case *ast.Ident:
		// panic("case 3")
		if targetsKind(deref, kinds...) {
			str, ok := dvals[0].Source().(fmt.Stringer)
			if !ok {
				panic("expected dvals[0].Source() to implement String()")
			}

			return tsast.SelectorExpr{
				Expr: ts.Ident(str.String()),
				Sel:  ts.Ident(dstr),
			}, nil
		}
	default:
		return nil, valError(v, "unknown selector subject type %T, cannot translate path %s", dvals[0].Source(), v.Path().String())
	}

	return nil, nil
}

func commentsForGroup(cgs []*ast.CommentGroup, jsdoc bool) []tsast.Comment {
	if cgs == nil {
		return nil
	}
	ret := make([]tsast.Comment, 0, len(cgs))
	for _, cg := range cgs {
		if cg.Line {
			panic("hit it")
		}
		ret = append(ret, ts.CommentFromCUEGroup(cg, jsdoc))
	}
	return ret
}

func commentsFor(v cue.Value, jsdoc bool) []tsast.Comment {
	return commentsForGroup(v.Doc(), jsdoc)
}
