package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"syscall/js"

	"github.com/grafana/thema"
	"github.com/grafana/thema/load"
	"github.com/grafana/thema/vmux"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"github.com/liamg/memoryfs"
)

var rt = thema.NewRuntime(cuecontext.New())

func main() {
	fmt.Println("Go Web Assembly")

	js.Global().Set("validateAny", runValidateAny())
	js.Global().Set("validateVersion", runValidateVersion())
	js.Global().Set("translateToLatest", runTranslateToLatest())
	js.Global().Set("getLineageVersions", runGetLineageVersions())
	js.Global().Set("format", runFormat())

	<-make(chan bool)
}

func runValidateVersion() js.Func {
	fn := js.FuncOf(func(this js.Value, args []js.Value) any {
		lineage := args[0].String()
		inputJSON := args[1].String()
		version := args[2].String()

		if lineage == "" || inputJSON == "" || version == "" {
			return toResult("", errors.New("lineage, input JSON or version is missing"))
		}

		datval, err := decodeData(inputJSON)
		if err != nil {
			return toResult("", err)
		}

		lin, err := loadLineage(lineage)
		if err != nil {
			return toResult("", err)
		}

		return toResult(validateVersion(lin, datval, version))
	})

	return fn
}

func runValidateAny() js.Func {
	fn := js.FuncOf(func(this js.Value, args []js.Value) any {
		lineage := args[0].String()
		inputJSON := args[1].String()

		if lineage == "" || inputJSON == "" {
			return toResult("", errors.New("lineage or input JSON is missing"))
		}

		datval, err := decodeData(inputJSON)
		if err != nil {
			return toResult("", err)
		}

		lin, err := loadLineage(lineage)
		if err != nil {
			return toResult("", err)
		}

		return toResult(validateAny(lin, datval))
	})

	return fn
}

func runTranslateToLatest() js.Func {
	fn := js.FuncOf(func(this js.Value, args []js.Value) any {
		lineage := args[0].String()
		inputJSON := args[1].String()

		if lineage == "" || inputJSON == "" {
			return toResult("", errors.New("lineage or input JSON is missing"))
		}

		datval, err := decodeData(inputJSON)
		if err != nil {
			return toResult("", err)
		}

		lin, err := loadLineage(lineage)
		if err != nil {
			return toResult("", err)
		}

		return toResult(translateToLatest(lin, datval))
	})

	return fn
}

func runGetLineageVersions() js.Func {
	fn := js.FuncOf(func(this js.Value, args []js.Value) any {
		lineage := args[0].String()

		if lineage == "" {
			return toResult("", errors.New("lineage is missing"))
		}

		lin, err := loadLineage(lineage)
		if err != nil {
			return toResult("", err)
		}

		return toResult(lineageVersions(lin))
	})

	return fn
}

func runFormat() js.Func {
	fn := js.FuncOf(func(this js.Value, args []js.Value) any {
		lineage := args[0].String()

		if lineage == "" {
			return toResult("", errors.New("lineage or input JSON is missing"))
		}

		inp := themaHeader + lineage
		res, err := format.Source([]byte(inp), format.TabIndent(true))
		if err != nil {
			return toResult("", err)
		}

		output := strings.TrimPrefix(string(res), themaHeader)
		return toResult(output, err)
	})

	return fn
}

func toResult(res any, err error) map[string]any {
	var errStr string
	if err != nil {
		errStr = fmt.Sprintf("%s", err)
	}
	return map[string]any{
		"result": res,
		"error":  errStr,
	}
}

const themaHeader = `package example

import "github.com/grafana/thema"

thema.#Lineage
name: "example"
`

func loadLineage(lineage string) (thema.Lineage, error) {
	fs := memoryfs.New()

	// Create cue.mod
	err := fs.MkdirAll("cue.mod", 0777)
	if err != nil {
		panic(err)
	}

	// Create module.cue
	err = fs.WriteFile("cue.mod/module.cue", []byte(`module: "github.com/grafana/example"`), 0777)
	if err != nil {
		return nil, err
	}

	err = fs.WriteFile("example.cue", []byte(themaHeader+lineage), 0777)
	if err != nil {
		return nil, err
	}

	inst, err := load.InstanceWithThema(fs, "")
	if err != nil {
		return nil, err
	}

	val := rt.Context().BuildInstance(inst)
	lin, err := thema.BindLineage(val, rt)
	if err != nil {
		return nil, err
	}

	return lin, nil
}

func decodeData(inputJSON string) (cue.Value, error) {
	if inputJSON == "" {
		return cue.Value{}, errors.New("data is missing")
	}

	jd := vmux.NewJSONCodec("stdin")
	datval, err := jd.Decode(rt.Underlying().Context(), []byte(inputJSON))
	if err != nil {
		return cue.Value{}, fmt.Errorf("failed to decode input data: %w", err)
	}
	return datval, nil
}

const latestVersion = "latest"

func validateVersion(lin thema.Lineage, datval cue.Value, version string) (string, error) {
	if !datval.Exists() {
		return "", errors.New("cue value does not exist")
	}

	var sch thema.Schema
	if version == latestVersion {
		sch = lin.Latest()
	} else {
		synv, err := thema.ParseSyntacticVersion(version)
		if err != nil {
			return "", err
		}
		sch, err = lin.Schema(synv)
		if err != nil {
			return "", fmt.Errorf("schema version %v does not exist in lineage", synv)
		}
	}

	_, err := sch.Validate(datval)
	if err != nil {
		return "", fmt.Errorf("input does not match version %s", version)
	}

	return fmt.Sprintf("input matches version %s", version), nil
}

func validateAny(lin thema.Lineage, datval cue.Value) (string, error) {
	if !datval.Exists() {
		return "", errors.New("cue value does not exist")
	}

	// TODO - is this needed?
	//var reterr error
	//if dc.lla.dl.sch != nil {
	//	_, reterr = dc.lla.dl.sch.Validate(dc.datval)
	//	if reterr == nil {
	//		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", dc.lla.dl.sch.Version())
	//		return nil
	//	}
	//}

	inst := lin.ValidateAny(datval)
	if inst != nil {
		return fmt.Sprintf("%s", inst.Schema().Version()), nil
	}

	//if reterr != nil {
	//	return "", reterr
	//}

	return "", errors.New("input does not match any version")
}

func translateToLatest(lin thema.Lineage, datval cue.Value) (string, error) {
	if !datval.Exists() {
		return "", errors.New("cue value does not exist")
	}

	inst := lin.ValidateAny(datval)
	if inst == nil {
		return "", errors.New("input data is not valid for any schema in lineage")
	}

	tinst, lac := inst.Translate(lin.Latest().Version())
	if tinst == nil {
		return "", errors.New("unreachable, thema.Translate() should never return a nil instance")
	}

	raw := tinst.Underlying()
	if !raw.Exists() {
		return "", errors.New("should be unreachable - result should at least always exist")
	}

	if raw.Err() != nil {
		return "", fmt.Errorf("translated value has errors, should be unreachable: %w", raw.Err())
	}

	if !raw.IsConcrete() {
		return "", errors.New("translated value is not concrete (TODO print non-concrete fields)")
	}

	// TODO support non-JSON output
	r := translationResult{
		From:    inst.Schema().Version().String(),
		To:      tinst.Schema().Version().String(),
		Result:  tinst.Underlying(),
		Lacunas: lac,
	}

	byt, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("error marshaling translation result to JSON: %w", err)
	}

	return string(byt), nil
}

type translationResult struct {
	From    string                   `json:"from"`
	To      string                   `json:"to,omitempty"`
	Result  cue.Value                `json:"result"`
	Lacunas thema.TranslationLacunas `json:"lacunas"`
}

func lineageVersions(lin thema.Lineage) (string, error) {
	ver := versions(lin.First(), []string{})
	byt, err := json.Marshal(ver)
	if err != nil {
		return "", fmt.Errorf("error marshaling versions result to JSON: %w", err)
	}

	return string(byt), nil
}

// versions walks the lineage from the first till the latest schema and adds their versions to a slice
func versions(sch thema.Schema, ver []string) []string {
	if sch == nil {
		return ver
	}

	ver = append(ver, sch.Version().String())

	return versions(sch.Successor(), ver)
}

//func runHydrate(lin thema.Lineage, datval cue.Value) (string, error) {
//	if !datval.Exists() {
//		panic("datval does not exist")
//	}
//
//	inst := lin.ValidateAny(datval)
//	if inst == nil {
//		return "", errors.New("input data is not valid for any schema in lineage")
//	}
//
//	byt, err := json.MarshalIndent(inst.Hydrate().Underlying(), "", "  ")
//	if err != nil {
//		return "", fmt.Errorf("error marshaling hydrated object to JSON: %w", err)
//	}
//	return string(byt), err
//}
