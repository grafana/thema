# Quickstart: Using Thema in Go

The tutorial demonstrates how a user can create an empty Lineage using the thema CLI, define a simple schema and use it to generate Go types and bindings, and use the bindings to validate data against a selected schema in the Lineage.

## Defining a schema
The most fundamental task in thema is writing a lineage. We can use the Thema CLI to generate an empty lineage using the below command:

```
thema lineage init empty --name ship > ship.cue 
```

This command will generate an empty lineage named `ship` and store the output in a file named `ship.cue`. The generated lineage looks like below

```cue
package ship

import "github.com/grafana/thema"

thema.#Lineage
name: "ship"
seqs: [
	{
		schemas: [
			// v0.0
			{
				// TODO (delete me - first schema goes here!)
			},
		]
	},
]
```

Let's define a simple schema as an object containing two fields named `name` of type `string` and `masts` of type `uint8` along with a constraint that masts are not more than 7,  

```cue
package ship

import "github.com/grafana/thema"

thema.#Lineage
name: "ship"
seqs: [
	{
		schemas: [
			// v0.0
			{
				// name is what we call the ship, and what's written in big letters on its hull
				name: string
				// masts is the number of masts the ship has. No fully rigged ship
				// has ever had more than 7: https://oceannavigator.com/the-most-masted-schooner-ever-built/
				masts: uint8 & < 8
			},
		]
	},
]
```

## Generating Go Types

Once we have a schema defined, We can use it to generate a type in Go. Use the below command to generate a Go type


```
thema lineage gen gotypes -l ship.cue > go_type.go 
```

This should generate a go file has a type `ship` defined . The generated file looks like below

```go
// This file is autogenerated. DO NOT EDIT.
//
// Generated by "thema lineage gen" from lineage defined in ship.cue

package ship

// Ship defines model for ship.
type Ship struct {
	// masts is the number of masts the ship has. No fully rigged ship
	// has ever had more than 7: https://oceannavigator.com/the-most-masted-schooner-ever-built/
	Masts int `json:"masts"`

	// name is what we call the ship, and what's written in big letters on its hull
	Name string `json:"name"`
}
```


CUE types are more expressive than Go types. To use the rich information from CUE in Go programs, users can use bindings. Go bindings provide access to the thema Lineage defined in `ship.cue`, and validate data against schemas in the lineage.

## Generating Go Bindings

Use the below command to generate a Go binding from `ship.cue`

```
thema lineage gen gobindings -l ship.cue > go_bindings.go
```

The generated Go bindings will look like this (generated comments have been removed):

```go
package ship

import (
	"embed"
	"path"

	"github.com/grafana/thema"
	"github.com/grafana/thema/load"
)

//go:embed ship.cue
var themaFS embed.FS

func Lineage(rt *thema.Runtime, opts ...thema.BindOption) (thema.Lineage, error) {
	// Load a build instance from the embedded fs
	inst, err := load.InstancesWithThema(themaFS, path.Dir("ship.cue"))
	if err != nil {
		return nil, err
	}

	raw := rt.Context().BuildInstance(inst)
	return thema.BindLineage(raw, rt)
}

// type guards

var _ thema.LineageFactory = Lineage
```
We now have a single Go function our program can call, and it will load up the `ship.cue` from disk and return a [`thema.Lineage`](https://pkg.go.dev/github.com/grafana/thema#Lineage)
We can then use the Lineage function to validate our data against the schema of choice

## Validating data

Using the Thema CLI, Users can validate the schema of choice defined in `ship.cue`. Use the below command to validate the schema against an invalid JSON data 

```
curl https://raw.githubusercontent.com/grafana/thema/main/docs/test_ship.json > test_ship.json
thema data validate -v 0.0 -l ship.cue test_ship.json
```

You should see an error similar to below
```
#ship00.masts: invalid value 9 (out of bound <8)
```

The Go Program will need `cue.mod` to be present, Run the below command to generate it

```
cue mod init
```

To validate the data from within a Go program, We can write a Go test for the below function. The function `validateInput` takes the input data and the schema the user wants to validate against


```go
func validateInput(input []byte, schema thema.SV(0,0)) ((thema.Instance, error)) {    
    ctx := cuecontext.New()
    lin, _ := Lineage(thema.NewRuntime(ctx))
    sch, _ := lin.Schema(schema)  //thema.SV(0,0) here represents first schema of first sequence
    expr, _ := json.Extract("input", input)
    val := ctx.BuildExpr(expr)

    return sch.Validate(val)
}
```

We can write a Go Test similar to below for the `validateInput` function

```go
func validateInput(t *testingT) {    
	var input = []byte(`{
        "name": "thema"
        masts: 9
    }`)
    _ , err := validateInput(input, thema.SV(0,0))

	if err != nil {
		fmt.Prntln(err)
	}
}
```

## Wrap up
This tutorial demonstrated how you can create an empty Lineage using the thema CLI, define a simple schema and use it to generate Go types and bindings, and use the bindings to validate data against a selected schema in the Lineage. 