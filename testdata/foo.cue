package schmoo

import "github.com/grafana/thema/testdata/internal:thema"

(thema.#Translate & {
	lin:  thema.basic
	inst: {
		all:  99999
		init: "i'mma string"
	}
	from: [0, 0]
	to: [1, 0]
}).out.result.result
