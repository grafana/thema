package exemplars

import "github.com/grafana/thema"

narrowing: {
    description: "Lineage that narrows a sloppily-specified boolean/string-ish type to a proper boolean over a single breaking change."
    l: thema.#Lineage & {
        seqs: [
            {
                schemas: [
                    {
                        boolish: "true" | "false" | bool | string
                    }
                ]
            },
            {
                schemas: [
                    {
                        properbool: bool
                    }
                ]

                lens: forward: {
                    to: seqs[1].schemas[0]
                    from: seqs[0].schemas[0]
                    translated: to & rel
                    rel: {
                        if ((from.boolish & string) != _|_) {
                            properbool: from.boolish == "true"
                        }
                        if ((from.boolish & bool) != _|_) {
                            properbool: from.boolish
                        }
                    }
                    lacunas: [
                        if ((from.boolish & string) != _|_) && ((from.boolish & ("true" | "false")) == _|_) {
                            thema.#Lacuna & {
                                sourceFields: [{
                                    path: "boolish"
                                    value: from.boolish
                                }]
                                targetFields: [{
                                    path: "properbool"
                                    value: to.properbool
                                }]
                                message: "boolish was a string but neither \"true\" nor \"false\"; fallback to treating as false"
                                type: thema.#LacunaTypes.LossyFieldMapping
                            }
                        }
                    ]
                }

                lens: reverse: {
                    to: seqs[0].schemas[0]
                    from: seqs[1].schemas[0]
                    translated: to & rel
                    rel: {
                        // Preserving preicse original form is a non-goal of thema in general.
                        boolish: from.properbool
                    }
                    lacunas: []
                }
            }
        ]
    }
}