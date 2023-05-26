package cuetil

import (
	"cuelang.org/go/cue"
)

func Subsume(val1 cue.Value, val2 cue.Value) error {
	err := val1.Subsume(val2, cue.Raw())
	if err != nil {
		// cueErr, ok := err.(errors.Error)
		// if !ok {
		// 	return err
		// }

		// errs := errors.Errors(cueErr)
		// var finalErr errors.Error
		// for _, e := range errs {
		// 	msg, args := e.Msg()

		// 	if msg == "invalid value %s (does not satisfy %s)" {
		// 		if args[0] == args[1] {
		// 			continue
		// 		}
		// 	}

		// 	finalErr = errors.Append(finalErr, e)
		// }

		// return finalErr
		return err
	}

	return nil
}

// Equal reports nil when the two cue values subsume each other or an error otherwise
func Equal(val1 cue.Value, val2 cue.Value) error {
	if err := Subsume(val1, val2); err != nil {
		return err
	}

	if err := Subsume(val2, val1); err != nil {
		return err
	}

	return nil
}
