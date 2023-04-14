package exemplars

disjunct: {
	description: "Changes over schemas with complex disjunctions."
	l: {
		schemas: [{
			version: [0, 0]
			schema: rootfield: {
				branch:    1
				branchone: string
			} | {
				branch:    2
				branchtwo: string
			}
		}, {
			version: [0, 1]
			schema: rootfield: {
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
		}]
		lenses: [{
			to: [0, 0]
			from: [0, 1]
			input:  _
			result: _|_ // TODO implement this lens
			lacunas: []
		}]
	}
}
