package exemplars

narrowing: {
    description: "Lineage that narrows a sloppily-specified boolean/string-ish type to a proper boolean over a single breaking change."
    l: {
        Seqs: [
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
                    to: lineages[1].schemas[0]
                    from: lineages[0].schemas[0]
                    out: to & rel
                    rel: {
                        if ((from.boolish & string) != _|_) {
                            properbool: from.boolish == "true"
                        }
                        if ((from.boolish & bool) != _|_) {
                            properbool: from.boolish
                        }
                    }
                    lacunae: [
                        if ((from.boolish & string) != _|_) && ((from.boolish & ("true" | "false")) == _|_) {
                            scuemata.#Lacuna & {
                                sourceFields: [{
                                    path: "boolish"
                                    value: from.boolish
                                }]
                                targetFields: [{
                                    path: "properbool"
                                    value: to.properbool
                                }]
                                message: "boolish was a string but neither \"true\" nor \"false\"; fallback to treating as false"
                                type: scuemata.#LacunaTypes.LossyFieldMapping
                            }
                        }
                    ]
                }

                lens: reverse: {
                    to: lineages[0].schemas[0]
                    from: lineages[1].schemas[0]
                    out: to & rel
                    rel: {
                        // Preserving preicse original form is a non-goal of scuemata in general.
                        boolish: from.properbool
                    }
                    lacunae: []
                }
            }
        ]
    }
}