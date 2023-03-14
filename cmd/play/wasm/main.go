package main

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/grafana/thema"
	"github.com/grafana/thema/load"
	"github.com/grafana/thema/vmux"
	"github.com/liamg/memoryfs"
	"syscall/js"
)

var rt = thema.NewRuntime(cuecontext.New())

type fn string

const validateAny fn = "validateAny"
const translateToLatest fn = "translateToLatest"
const linVersions fn = "linVersions"

//const hydrate fn = "hydrade"

func main() {
	fmt.Println("Go Web Assembly")
	js.Global().Set(string(validateAny), wrap(validateAny))
	js.Global().Set(string(translateToLatest), wrap(translateToLatest))
	js.Global().Set(string(linVersions), wrap(linVersions))
	<-make(chan bool)
}

func wrap(action fn) js.Func {
	fn := js.FuncOf(func(this js.Value, args []js.Value) any {
		lineage := args[0].String()
		version := args[1].String()
		data := args[2].String()

		res, err := handle(action, lineage, version, data)
		//fmt.Println("Result from Go:", res)
		var errStr string
		if err != nil {
			errStr = fmt.Sprintf("%s failed: %s\n", action, err)
		}
		return map[string]any{
			"result": res,
			"error":  errStr,
		}
	})

	return fn
}

const lineageHeader = `package example
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

	err = fs.WriteFile("example.cue", []byte(lineageHeader+lineage), 0777)
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

func handle(action fn, lineage string, version string, data string) (string, error) {
	if lineage == "" {
		return "", errors.New("lineage is missing")
	}

	lin, err := loadLineage(lineage)
	if err != nil {
		return "", err
	}

	switch action {
	case validateAny:
		datval, err := decodeData(data)
		if err != nil {
			return "", err
		}
		return runValidateAny(lin, datval)
	case translateToLatest:
		datval, err := decodeData(data)
		if err != nil {
			return "", err
		}
		return runTranslateToLatest(lin, datval)
	case linVersions:
		return lineageVersions(lin)
	//case hydrate:
	//	return runHydrate(lin, datval)
	default:
		return "", fmt.Errorf("undefined action")
	}

}

func decodeData(data string) (cue.Value, error) {
	if data == "" {
		return cue.Value{}, errors.New("data is missing")
	}

	jd := vmux.NewJSONCodec("stdin")
	datval, err := jd.Decode(rt.Underlying().Context(), []byte(data))
	if err != nil {
		return cue.Value{}, err
	}
	return datval, err
}

func runValidateAny(lin thema.Lineage, datval cue.Value) (string, error) {
	if !datval.Exists() {
		panic("datval does not exist")
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
		return fmt.Sprintf("%s\n", inst.Schema().Version()), nil
	}

	//if reterr != nil {
	//	return "", reterr
	//}

	return "", fmt.Errorf("does not match any version")
}

func runTranslateToLatest(lin thema.Lineage, datval cue.Value) (string, error) {
	if !datval.Exists() {
		panic("datval does not exist")
	}

	inst := lin.ValidateAny(datval)
	if inst == nil {
		return "", errors.New("input data is not valid for any schema in lineage")
	}

	tinst, lac := inst.Translate(lin.Latest().Version())
	if tinst == nil {
		panic("unreachable, thema.Translate() should never return a nil instance")
	}

	raw := tinst.Underlying()
	if !raw.Exists() {
		return "", errors.New("should be unreachable - result should at least always exist")
	}

	if raw.Err() != nil {
		return "", fmt.Errorf("translated value has errors, should be unreachable: %w", raw.Err())
	}

	if !raw.IsConcrete() {
		return "", fmt.Errorf("translated value is not concrete (TODO print non-concrete fields)")
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
