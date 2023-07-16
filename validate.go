package thema

import (
	"bytes"
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/token"

	terrors "github.com/grafana/thema/errors"
)

type onesidederr struct {
	schpos, datapos []token.Pos
	code            terrors.ValidationCode
	coords          coords
	val             string
}

func (e *onesidederr) Error() string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "%s: validation failed, data is not an instance:", e.coords)
	switch e.code {
	case terrors.MissingField:
		fmt.Fprintf(&buf, "\n\tschema specifies that field exists with type %v", e.val)
		for _, pos := range e.schpos {
			fmt.Fprintf(&buf, "\n\t\t%s", pos.String())
		}

		fmt.Fprintf(&buf, "\n\tbut field was absent from data")
		for _, pos := range e.datapos {
			fmt.Fprintf(&buf, "\n\t\t%s", pos.String())
		}
	case terrors.ExcessField:
		fmt.Fprintf(&buf, "\n\tschema is closed and does not specify field")
		for _, pos := range e.schpos {
			fmt.Fprintf(&buf, "\n\t\t%s", pos.String())
		}

		fmt.Fprintf(&buf, "\n\tbut field exists in data with value %v", e.val)
		for _, pos := range e.datapos {
			fmt.Fprintf(&buf, "\n\t\t%s", pos.String())
		}
	}

	return buf.String()
}

func (e *onesidederr) Unwrap() error {
	return terrors.ErrInvalidData
}

type twosidederr struct {
	schpos, datapos []token.Pos
	code            terrors.ValidationCode
	coords          coords
	sv, dv          string
}

func (e *twosidederr) Error() string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "%s: validation failed, data is not an instance:\n\tschema expected `%s`", e.coords, e.sv)
	for _, pos := range e.schpos {
		fmt.Fprintf(&buf, "\n\t\t%s", pos.String())
	}

	fmt.Fprintf(&buf, "\n\tbut data contained `%s`", e.dv)
	for _, pos := range e.datapos {
		fmt.Fprintf(&buf, "\n\t\t%s", pos.String())
	}
	return buf.String()
}

func (e *twosidederr) Unwrap() error {
	return terrors.ErrInvalidData
}

// TODO differentiate this once we have generic composition to support trimming out irrelevant disj branches
type emptydisjunction struct {
	schpos, datapos []token.Pos
	coords          coords
	brancherrs      []error
}

func (e *emptydisjunction) Unwrap() error {
	return terrors.ErrInvalidData
}

type validationFailure []error

func (vf validationFailure) Unwrap() error {
	return terrors.ErrInvalidData
}

func (vf validationFailure) Error() string {
	var buf bytes.Buffer
	for _, e := range vf {
		fmt.Fprint(&buf, e.Error())
		fmt.Fprintf(&buf, "\n")
	}

	return buf.String()
}

// HERE BE DRAGONS, BRING A SWORD.
func mungeValidateErr(err error, sch Schema) error {
	_, is := err.(errors.Error)
	if !is {
		return err
	}

	var errs validationFailure
	e := errors.Errors(err)
	for _, ee := range e {
		schpos, datapos := splitTokens(ee.InputPositions())
		x := coords{
			sch:       sch,
			fieldpath: trimThemaPath(ee.Path()),
		}

		msg, vals := ee.Msg()
		switch len(vals) {
		case 1:
			val, ok := vals[0].(string)
			if !ok {
				break
			}
			err := &onesidederr{
				schpos:  schpos,
				datapos: datapos,
				coords:  x,
				val:     val,
			}

			if strings.Contains(msg, "incomplete") {
				err.code = terrors.MissingField
			} else if strings.Contains(msg, "not allowed") {
				err.code = terrors.ExcessField
			} else {
				break
			}

			errs = append(errs, err)
			continue
		case 2:
			dataval, dvok := vals[0].(string)
			schval, svok := vals[1].(string)
			if !svok || !dvok {
				break
			}

			errs = append(errs, &twosidederr{
				schpos:  schpos,
				datapos: datapos,
				coords:  x,
				sv:      schval,
				dv:      dataval,
				code:    terrors.OutOfBounds,
			})
			continue

		case 4:
			schval, svok := vals[0].(string)
			dataval, dvok := vals[1].(string)
			schkind, skok := vals[2].(cue.Kind)
			datakind, dkok := vals[3].(cue.Kind)

			// if type is in map, then it's a schval, not dataval
			// todo: this needs to be combined somehow with L184-187
			if m, ok := schErrMsgFormatMap[dataval]; ok {
				dataval = schval
				schval = m
				schkind, datakind = datakind, schkind
			}

			if !svok || !dvok || !skok || !dkok {
				break
			}

			if schkind&cue.NumberKind > 0 {
				if m, ok := schErrMsgFormatMap[schval]; ok {
					schval = m
				}
			}

			err := &twosidederr{
				schpos:  schpos,
				datapos: datapos,
				coords:  x,
				sv:      schval,
				dv:      dataval,
			}
			if datakind.IsAnyOf(schkind) {
				err.code = terrors.OutOfBounds
			} else {
				err.code = terrors.KindConflict
			}

			errs = append(errs, err)
			continue
		}
	}
	return errs
}

var schErrMsgFormatMap = map[string]string{
	"int & >=-2147483648 & <=2147483647":                                                                   "int32",
	"int & >=-9223372036854775808 & <=9223372036854775807":                                                 "int64",
	">=-340282346638528859811704183484516925440 & <=340282346638528859811704183484516925440":               "float32",
	">=-1.797693134862315708145274237317043567981E+308 & <=1.797693134862315708145274237317043567981E+308": "float64",
}

func splitTokens(poslist []token.Pos) (schpos, datapos []token.Pos) {
	if len(poslist) == 0 {
		return
	}

	// We're assuming data is always last. ...Probably safe? Given that we
	// control the order of operands in the Schema.Validate() calls...
	dataname := poslist[len(poslist)-1].Filename()
	var split int
	for i, pos := range poslist {
		if pos.Filename() == dataname {
			split = i
			break
		}
	}

	return poslist[:split], poslist[split:]
}

func trimThemaPath(parts []string) []string {
	for i, s := range parts {
		if s == "schemas" {
			return parts[i+3:]
		}
	}

	// Otherwise, it's one of the defpath patterns - eliminate first element
	return parts[1:]
}

type coords struct {
	sch       Schema
	fieldpath []string
}

func (c coords) String() string {
	return fmt.Sprintf("<%s@v%s>.%s", c.sch.Lineage().Name(), c.sch.Version(), strings.Join(c.fieldpath, "."))
}
