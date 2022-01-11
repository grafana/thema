package exemplars

import (
	"embed"
)

// cueFS contains the raw .cue files with all the thema exemplars.
//
//go:embed *.cue
var cueFS embed.FS
