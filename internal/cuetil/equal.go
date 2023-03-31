package cuetil

import (
	"regexp"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
)

func toString(val cue.Value) (string, error) {
	node := val.Syntax(
		cue.All(),
		cue.Raw(),
		cue.Schema(),
		cue.Definitions(true),
		cue.Docs(true),
		cue.Hidden(true),
	)

	bytes, err := format.Node(node)
	if err != nil {
		return "", err
	}

	regWhitespace := regexp.MustCompile("\\s+")
	res := regWhitespace.ReplaceAllString(string(bytes), "")
	
	regDef := regexp.MustCompile(`{_#def_#def:(.*)}`)
	matches := regDef.FindStringSubmatch(res)
	if len(matches) > 1 {
		res = matches[1]
	} else {
		res = strings.Replace(res, "_#def_#def:", "", 1)
	}

	return res, nil
}

func Equal(val1 cue.Value, val2 cue.Value) bool {
	string1, err := toString(val1)
	if err != nil {
		return false
	}

	string2, err := toString(val2)
	if err != nil {
		return false
	}
	
	return len(string1) == len(string2) && string1 == string2
}