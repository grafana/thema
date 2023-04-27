package cuetil

import (
	"cuelang.org/go/cue"
)

func Equal(val1 cue.Value, val2 cue.Value) error {
	if err := val1.Subsume(val2, cue.Raw(), cue.Schema(), cue.Definitions(true), cue.All(), cue.Final()); err != nil {
		return err
	}

	if err := val2.Subsume(val1, cue.Raw(), cue.Schema(), cue.Definitions(true), cue.All(), cue.Final()); err != nil {
		return err
	}

	return nil
}
