package exemplars

disjunct: {
	description: "Changes over schemas with complex disjunctions."
	l: {
		seqs: [
			{
				schemas: [
					{// 0.0
						rootfield: {
							branch:    1
							branchone: string
						} | {
							branch:    2
							branchtwo: string
						}
					},
					{// 0.1
						rootfield: {
							branch:    1
							branchone: string
						} | {
							branch:    2
							branchtwo: string
						} | {
							branch:      3
							branchthree: {
								branchthreeinner: int
							} | {
								branchthreeinner: string
							}
						}
					},
				]
			},
		]
	}
}