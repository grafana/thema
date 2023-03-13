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
const hydrate fn = "hydrade"
const translateToLatest fn = "translateToLatest"

func main() {
	fmt.Println("Go Web Assembly")
	js.Global().Set(string(validateAny), wrapValidate(validateAny))
	js.Global().Set(string(translateToLatest), wrapValidate(translateToLatest))
	<-make(chan bool)
}

func wrapValidate(action fn) js.Func {
	fn := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 3 {
			result := map[string]any{
				"error": "Invalid no of arguments passed",
			}
			return result
		}

		jsDoc := js.Global().Get("document")
		if !jsDoc.Truthy() {
			result := map[string]any{
				"error": "Unable to get document object",
			}
			return result
		}

		jsOutput := jsDoc.Call("getElementById", "output")
		if !jsOutput.Truthy() {
			result := map[string]any{
				"error": "Unable to get output text area",
			}
			return result
		}
		//fmt.Printf("args %s\n", args)

		lineage := args[0].String()
		version := args[1].String()
		data := args[2].String()

		res, err := handle(action, lineage, version, data)
		if err != nil {
			errStr := fmt.Sprintf("%s failed: %s\n", action, err)
			result := map[string]any{
				"error": errStr,
			}
			return result
		}

		// TODO: delete it later
		if res == "" {
			res = "action output is empty, so probably success"
		}
		jsOutput.Set("value", res)
		return nil
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
	lin, err := loadLineage(lineage)
	if err != nil {
		return "", err
	}

	jd := vmux.NewJSONCodec("stdin")
	datval, err := jd.Decode(rt.Underlying().Context(), []byte(data))
	if err != nil {
		return "", err
	}

	switch action {
	case validateAny:
		return runValidateAny(lin, version, datval)
	case translateToLatest:
		return runTranslateToLatest(lin, version, datval)
	//case hydrate:
	//	return runHydrate(lin, datval)
	default:
		return "", fmt.Errorf("undefined action")
	}

}

func runValidateAny(lin thema.Lineage, _ string, datval cue.Value) (string, error) {
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

func runTranslateToLatest(lin thema.Lineage, _ string, datval cue.Value) (string, error) {
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

	byt, err := json.MarshalIndent(r, "", "  ")
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

func runHydrate(lin thema.Lineage, datval cue.Value) (string, error) {
	if !datval.Exists() {
		panic("datval does not exist")
	}

	inst := lin.ValidateAny(datval)
	if inst == nil {
		return "", errors.New("input data is not valid for any schema in lineage")
	}

	byt, err := json.MarshalIndent(inst.Hydrate().Underlying(), "", "  ")
	if err != nil {
		return "", fmt.Errorf("error marshaling hydrated object to JSON: %w", err)
	}
	return string(byt), err
}
