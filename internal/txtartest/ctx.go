package txtartest

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/grafana/thema"
)

var _ctx *cue.Context
var _rt *thema.Runtime

func init() {
	_ctx = cuecontext.New()
	_rt = thema.NewRuntime(_ctx)
}

// Context returns the central context used by default for testing.
func Context() *cue.Context{
	return _ctx
}

// Runtime returns the central runtime used by default for testing.
func Runtime() *thema.Runtime {
	return _rt
}
