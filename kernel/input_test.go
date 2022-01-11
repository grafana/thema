package kernel

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/grafana/thema"
	"github.com/grafana/thema/exemplars"
)

var jsondl = NewJSONDecoder("test")

type type00 struct {
	Before    string `json:"before"`
	Unchanged string `json:"unchanged"`
}
type type10 struct {
	After     string `json:"after"`
	Unchanged string `json:"unchanged"`
}

var tf00 = func() interface{} {
	return type00{}
}
var tf10 = func() interface{} {
	return type10{}
}

func TestInputKernelInputs(t *testing.T) {
	ctx := cuecontext.New()
	lib := thema.NewLibrary(ctx)

	lin, err := exemplars.RenameLineage(lib)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("missing-lineage-panic", func(t *testing.T) {
		cfg := InputKernelConfig{
			Loader:      jsondl,
			TypeFactory: tf10,
		}
		defer func() {
			recover()
		}()

		NewInputKernel(cfg)
		t.Fatal("should panic")
	})

	t.Run("missing-loader-panic", func(t *testing.T) {
		cfg := InputKernelConfig{
			TypeFactory: tf10,
			Lineage:     lin,
		}
		defer func() {
			recover()
		}()

		NewInputKernel(cfg)
		t.Fatal("should panic")
	})

	t.Run("missing-tf-panic", func(t *testing.T) {
		cfg := InputKernelConfig{
			Loader:  jsondl,
			Lineage: lin,
		}
		defer func() {
			recover()
		}()

		NewInputKernel(cfg)
		t.Fatal("should panic")
	})

	t.Run("err-non-pointer-tf", func(t *testing.T) {
		cfg := InputKernelConfig{
			Loader: jsondl,
			TypeFactory: func() interface{} {
				return &type00{}
			},
			Lineage: lin,
			To:      thema.SV(0, 0),
		}
		_, err := NewInputKernel(cfg)
		if err == nil {
			t.Fatal("should fail when pointer type is emitted from type factory")
		}
	})

	t.Run("invalid-type", func(t *testing.T) {
		cfg := InputKernelConfig{
			Loader:      jsondl,
			TypeFactory: tf00,
			Lineage:     lin,
			To:          thema.SV(1, 0),
		}
		_, err := NewInputKernel(cfg)
		if err == nil {
			t.Fatal("should fail when type incompatible with schema is emitted from type factory")
		}

		cfg.To = thema.SV(0, 0)
		cfg.TypeFactory = tf10
		_, err = NewInputKernel(cfg)
		if err == nil {
			t.Fatal("should fail when type incompatible with schema is emitted from type factory")
		}
	})
}
