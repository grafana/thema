package exemplars

import (
	"embed"
)

// CueFS contains the raw .cue files with all the thema exemplars.
//
//go:embed *.cue
var CueFS embed.FS
