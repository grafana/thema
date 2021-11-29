package invalid

import "github.com/grafana/scuemata"

_#Harness: {
    l: scuemata.#Lineage
    failMessage: string
}

_#simpleSchema: {
    ssField: string
}

_#simpleSequence: {
    schemas: [_#simpleSchema]
}

[N=string]: _#Harness

empty: {
    l: {
        Name: "empty"
        Seqs: []
    }
}

nameless: {
    l: {
        Seqs: [_#simpleSequence]
    }
}

nolens: {
    l: {
        Seqs: [
            _#simpleSequence,
            {
                schemas: [
                    {
                        newField: string
                    }
                ]
                lens: {}
            }
        ]
    }
}