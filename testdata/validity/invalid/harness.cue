package invalid

import "github.com/grafana/thema"

_#Harness: {
    l: thema.#Lineage
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