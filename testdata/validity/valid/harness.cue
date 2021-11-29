package invalid

import "github.com/grafana/scuemata"

#Harness: {
    l: scuemata.#Lineage
    failMessage: string
}

[string]: #Harness

foo: {}