package main

import (
	"cuelang.org/go/cue/cuecontext"
	"fmt"
	"github.com/grafana/thema"
	"github.com/grafana/thema/load"
	"github.com/grafana/thema/vmux"
	"github.com/liamg/memoryfs"
)

var rt = thema.NewRuntime(cuecontext.New())

func main() {
	fmt.Println("Go Web Assembly")
	fmt.Println(rt.Underlying().String())

	//js.Global().Set("validate", wrapValidate())
	<-make(chan bool)
}

//func wrapValidate() js.Func {
//	fn := js.FuncOf(func(this js.Value, args []js.Value) any {
//		if len(args) != 3 {
//			result := map[string]any{
//				"error": "Invalid no of arguments passed",
//			}
//			return result
//		}
//
//		jsDoc := js.Global().Get("document")
//		if !jsDoc.Truthy() {
//			result := map[string]any{
//				"error": "Unable to get document object",
//			}
//			return result
//		}
//
//		jsOutput := jsDoc.Call("getElementById", "output")
//		if !jsOutput.Truthy() {
//			result := map[string]any{
//				"error": "Unable to get output text area",
//			}
//			return result
//		}
//		fmt.Printf("args %s\n", args)
//
//		lineage := args[0].String()
//		fmt.Printf("input lineage %s\n", lineage)
//
//		version := args[1].String()
//		fmt.Printf("input version %s\n", version)
//
//		data := args[2].String()
//		fmt.Printf("input data %s\n", data)
//
//		result, err := validate(lineage, version, data)
//		if err != nil {
//			errStr := fmt.Sprintf("validation failed: %s\n", err)
//			result := map[string]any{
//				"error": errStr,
//			}
//			return result
//		}
//
//		jsOutput.Set("value", result)
//		return nil
//	})
//
//	return fn
//}

func loadLineage(lineage []byte) (thema.Lineage, error) {
	fs := memoryfs.New()

	// Create cue.mod
	err := fs.MkdirAll("cue.mod", 0777)
	if err != nil {
		panic(err)
	}

	// Create module.cue
	err = fs.WriteFile("cue.mod/module.cue", []byte(`module: "github.com/grafana/ship"`), 0777)
	if err != nil {
		return nil, err
	}

	err = fs.WriteFile("ship.cue", lineage, 0777)
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

func validate(lineage string, version string, data string) (string, error) {
	lin, err := loadLineage([]byte(lineage))
	if err != nil {
		return "", err
	}

	jd := vmux.NewJSONCodec("stdin")
	datval, err := jd.Decode(rt.Underlying().Context(), []byte(data))
	if err != nil {
		return "", err
	}

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
