package thema

import (
	"bytes"
	"errors"

	"cuelang.org/go/cue"
	cuejson "cuelang.org/go/pkg/encoding/json"
)

// applyDefaults returns a new, concrete copy of the Resource with all paths
// that are 1) missing in the Resource AND 2) specified by the schema,
// filled with default values specified by the schema.
func applyDefaults(r Instance, scue cue.Value) (Instance, error) {
	rvUnified, err := applyDefaultHelper(r.raw, scue)
	if err != nil {
		return r, err
	}

	// re, err := convertCUEValueToString(rvUnified)
	// if err != nil {
	// 	return r, err
	// }

	return Instance{raw: rvUnified}, nil
}

func applyDefaultHelper(input cue.Value, scue cue.Value) (cue.Value, error) {
	switch scue.IncompleteKind() {
	case cue.ListKind:
		// if list element exist
		ele := scue.LookupPath(cue.MakePath(cue.AnyIndex))

		// if input is not a concrete list, we must have list elements exist to be used to trim defaults
		if ele.Exists() {
			if ele.IncompleteKind() == cue.BottomKind {
				return input, errors.New("can't get the element of list")
			}
			iter, err := input.List()
			if err != nil {
				return input, errors.New("can't apply defaults for list")
			}
			var iterlist []cue.Value
			for iter.Next() {
				ref, err := getBranch(ele, iter.Value())
				if err != nil {
					return input, err
				}
				re, err := applyDefaultHelper(iter.Value(), ref)
				if err == nil {
					iterlist = append(iterlist, re)
				}
			}
			liInstance := scue.Context().NewList(iterlist...)
			if liInstance.Err() != nil {
				return input, liInstance.Err()
			}
			return liInstance, nil
		}
		return input.Unify(scue), nil
	case cue.StructKind:
		iter, err := scue.Fields(cue.Optional(true))
		if err != nil {
			return input, err
		}
		for iter.Next() {
			lable, _ := iter.Value().Label()
			lv := input.LookupPath(cue.MakePath(cue.Str(lable)))
			if err != nil {
				continue
			}
			if lv.Exists() {
				res, err := applyDefaultHelper(lv, iter.Value())
				if err != nil {
					continue
				}
				input = input.FillPath(cue.MakePath(cue.Str(lable)), res)
			} else if !iter.IsOptional() {
				input = input.FillPath(cue.MakePath(cue.Str(lable)), iter.Value().Eval())
			}
		}
		return input, nil
	default:
		input = input.Unify(scue)
	}
	return input, nil
}

func convertCUEValueToString(inputCUE cue.Value) (string, error) {
	re, err := cuejson.Marshal(inputCUE)
	if err != nil {
		return re, err
	}

	result := []byte(re)
	result = bytes.Replace(result, []byte("\\u003c"), []byte("<"), -1)
	result = bytes.Replace(result, []byte("\\u003e"), []byte(">"), -1)
	result = bytes.Replace(result, []byte("\\u0026"), []byte("&"), -1)
	return string(result), nil
}

// trimDefaults returns a new, concrete copy of the Resource where all paths
// in the  where the values at those paths are the same as the default value
// given in the schema.
func trimDefaults(r Instance, scue cue.Value) (Instance, error) {
	rv, _, err := removeDefaultHelper(scue, r.raw)
	if err != nil {
		return r, err
	}

	// re, err := convertCUEValueToString(rv)
	// if err != nil {
	// 	return r, err
	// }

	return Instance{raw: rv}, nil
}

func getDefault(icue cue.Value) (cue.Value, bool) {
	d, exist := icue.Default()
	if exist && d.Kind() == cue.ListKind {
		len, err := d.Len().Int64()
		if err != nil {
			return d, false
		}
		var defaultExist bool
		if len <= 0 {
			op, vals := icue.Expr()
			if op == cue.OrOp {
				for _, val := range vals {
					vallen, _ := val.Len().Int64()
					if val.Kind() == cue.ListKind && vallen <= 0 {
						defaultExist = true
						break
					}
				}
				if !defaultExist {
					exist = false
				}
			} else {
				exist = false
			}
		}
	}
	return d, exist
}

func isCueValueEqual(inputdef cue.Value, input cue.Value) bool {
	d, exist := getDefault(inputdef)
	if exist {
		return input.Subsume(d) == nil && d.Subsume(input) == nil
	}
	return false
}

func removeDefaultHelper(inputdef cue.Value, input cue.Value) (cue.Value, bool, error) {
	// To include all optional fields, we need to use inputdef for iteration,
	// since the lookuppath with optional field doesn't work very well
	rv := inputdef.Context().CompileString("", cue.Filename("helper"))
	if rv.Err() != nil {
		return input, false, rv.Err()
	}

	switch inputdef.IncompleteKind() {
	case cue.StructKind:
		// Get all fields including optional fields
		iter, err := inputdef.Fields(cue.Optional(true))
		if err != nil {
			return rv, false, err
		}
		keySet := make(map[string]bool)
		for iter.Next() {
			lable, _ := iter.Value().Label()
			keySet[lable] = true
			lv := input.LookupPath(cue.MakePath(cue.Str(lable)))
			if err != nil {
				return rv, false, err
			}
			if lv.Exists() {
				re, isEqual, err := removeDefaultHelper(iter.Value(), lv)
				if err == nil && !isEqual {
					rv = rv.FillPath(cue.MakePath(cue.Str(lable)), re)
				}
			}
		}
		// Get all the fields that are not defined in schema yet for panel
		iter, err = input.Fields()
		if err != nil {
			return rv, false, err
		}
		for iter.Next() {
			lable, _ := iter.Value().Label()
			if exists := keySet[lable]; !exists {
				rv = rv.FillPath(cue.MakePath(cue.Str(lable)), iter.Value())
			}
		}
		return rv, false, nil
	case cue.ListKind:
		if isCueValueEqual(inputdef, input) {
			return rv, true, nil
		}

		// take every element of the list
		ele := inputdef.LookupPath(cue.MakePath(cue.AnyIndex))

		// if input is not a concrete list, we must have list elements exist to be used to trim defaults
		if ele.Exists() {
			if ele.IncompleteKind() == cue.BottomKind {
				return rv, true, nil
			}

			iter, err := input.List()
			if err != nil {
				return rv, true, nil
			}
			var iterlist []cue.Value
			for iter.Next() {
				ref, err := getBranch(ele, iter.Value())
				if err != nil {
					iterlist = append(iterlist, iter.Value())
					continue
				}
				re, isEqual, err := removeDefaultHelper(ref, iter.Value())
				if err == nil && !isEqual {
					iterlist = append(iterlist, re)
				} else {
					iterlist = append(iterlist, iter.Value())
				}
			}
			liInstance := inputdef.Context().NewList(iterlist...)
			return liInstance, false, liInstance.Err()
		}
		// now when ele is empty, we don't trim anything
		return input, false, nil

	default:
		if isCueValueEqual(inputdef, input) {
			return input, true, nil
		}
		return input, false, nil
	}
}

func getBranch(schemaObj cue.Value, concretObj cue.Value) (cue.Value, error) {
	op, defs := schemaObj.Expr()
	if op == cue.OrOp {
		for _, def := range defs {
			err := def.Unify(concretObj).Validate(cue.Concrete(true))
			if err == nil {
				return def, nil
			}
		}
		// no matching branches? wtf
		return schemaObj, errors.New("no branch is found for list")
	}
	return schemaObj, nil
}
