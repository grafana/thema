package main

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"fmt"
	"github.com/grafana/thema"
	"github.com/grafana/thema/vmux"
	"syscall/js"
)

func main() {
	fmt.Println("Go Web Assembly")
	var ctx = cuecontext.New()
	var rt = thema.NewRuntime(ctx)
	fmt.Println(rt.Underlying().String())
	//js.Global().Set("validate", wrapValidate())
	<-make(chan bool)
}

func wrapValidate() js.Func {
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
		fmt.Printf("args %s\n", args)

		lineage := args[0].String()
		fmt.Printf("input lineage %s\n", lineage)

		version := args[1].String()
		fmt.Printf("input version %s\n", version)

		data := args[2].String()
		fmt.Printf("input data %s\n", data)

		result, err := validate(lineage, version, data)
		if err != nil {
			errStr := fmt.Sprintf("validation failed: %s\n", err)
			result := map[string]any{
				"error": errStr,
			}
			return result
		}

		jsOutput.Set("value", result)
		return nil
	})

	return fn
}

func validate(lineagePath string, version string, data string) (string, error) {
	lla := new(lineageLoadArgs)
	var datval cue.Value

	// TODO: make it use bytes, not file path
	lla.inputLinFilePath = lineagePath
	lla.verstr = version

	err := lla.validateLineageInput(nil, nil)
	if err != nil {
		return "", fmt.Errorf("failed to validate lineagePath input: %v", err)
	}

	jd := vmux.NewJSONCodec("stdin")
	datval, err = jd.Decode(rt.Underlying().Context(), []byte(data))
	if err != nil {
		return "", err
	}

	if !datval.Exists() {
		panic("datval does not exist")
	}

	_, err = lla.dl.sch.Validate(datval)
	if err != nil {
		return "", err
	}

	return "", nil
}
