package invalid

import "github.com/grafana/thema"

#Harness: {
    l: thema.#Lineage
    failMessage: string
}

[string]: #Harness

foo: {}