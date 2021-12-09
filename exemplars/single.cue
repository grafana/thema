package exemplars

single: {
    description: "Lineage containing one sequence with a single, trivial schema."
    l: {
        Seqs: [
            {
                schemas: [
                    {
                        astring: string
                        anint: int
                        abool: bool
                    }
                ]
            }
        ]
    }
}