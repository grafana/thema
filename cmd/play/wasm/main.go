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

const fn_validate fn = "validate"
const fn_hydrate fn = "hydrade"

func main() {
	fmt.Println("Go Web Assembly")
	js.Global().Set(string(fn_validate), wrapValidate(fn_validate))
	js.Global().Set(string(fn_hydrate), wrapValidate(fn_hydrate))
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
	case fn_validate:
		return runValidate(lin, version, datval)
	case fn_hydrate:
		return runHydrate(lin, datval)
	default:
		return "", fmt.Errorf("undefined action")
	}

}

func runValidate(lin thema.Lineage, version string, datval cue.Value) (string, error) {
	if !datval.Exists() {
		panic("datval does not exist")
	}

	synv, err := thema.ParseSyntacticVersion(version)
	if err != nil {
		return "", err
	}

	sch, err := lin.Schema(synv)
	if err != nil {
		return "", err
	}

	_, err = sch.Validate(datval)
	if err != nil {
		return "", err
	}

	return "", nil
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
