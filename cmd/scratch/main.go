package main

import (
	"cuelang.org/go/cue/cuecontext"
	"fmt"
	"github.com/grafana/thema"
	"github.com/grafana/thema/load"
	"github.com/grafana/thema/vmux"
	"github.com/liamg/memoryfs"
)

var inputVal = `package ship
import "github.com/grafana/thema"
thema.#Lineage
name: "ship"
seqs: [
	{
		schemas: [
			// v0.0
			{
				field1: string
			},
		]
	},
]`

func main() {
	fs := memoryfs.New()

	// Create cue.mod
	err := fs.MkdirAll("cue.mod", 0777)
	if err != nil {
		panic(err)
	}

	// Create module.cue
	err = fs.WriteFile("cue.mod/module.cue", []byte(`module: "github.com/grafana/ship"`), 0777)
	if err != nil {
		panic(err)
	}

	err = fs.WriteFile("ship.cue", []byte(inputVal), 0777)
	if err != nil {
		panic(err)
	}

	inst, err := load.InstanceWithThema(fs, "")
	if err != nil {
		panic(err)
	}

	rt := thema.NewRuntime(cuecontext.New())
	val := rt.Context().BuildInstance(inst)

	lin, err := thema.BindLineage(val, rt)
	if err != nil {
		panic(err)
	}

	jd := vmux.NewJSONCodec("stdin")
	datval, err := jd.Decode(rt.Underlying().Context(), []byte(`{"field1": "100"}`))
	if err != nil {
		panic(err)
	}

	if !datval.Exists() {
		panic("datval does not exist")
	}

	_, err = lin.Latest().Validate(datval)

	fmt.Println(err)

	//fmt.Println("Go Web Assembly")
	//var ctx = cuecontext.New()
	//var rt = thema.NewRuntime(ctx)
	//
	//fmt.Println(rt.Underlying().String())
	////js.Global().Set("validate", wrapValidate())
	//<-make(chan bool)
}
