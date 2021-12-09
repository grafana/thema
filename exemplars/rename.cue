package exemplars

rename: {
    description: "A field is renamed - a breaking change, necessitating a new sequence."
    l: {
        Seqs: [
            {
                schemas: [
                    {
                        before: string
                        unchanged: string
                    }
                ]
            },
            {
                schemas: [
                    {
                        after: string
                        unchanged: string
                    }
                ]

                lens: forward: {
                    to: Seqs[1].schemas[0]
                    from: Seqs[0].schemas[0]
                    out: to & rel
                    rel: {
                        after: from.before
                        unchanged: from.unchanged
                    }
                    lacunae: []
                }
                lens: reverse: {
                    to: Seqs[0].schemas[0]
                    from: Seqs[1].schemas[0]
                    out: to & rel
                    rel: {
                        before: from.after
                        unchanged: from.unchanged
                    }
                    lacunae: []
                }
            }
        ]
    }
}