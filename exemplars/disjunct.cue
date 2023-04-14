package exemplars

disjunct: {
	description: "Changes over schemas with complex disjunctions."
	l: {
		schemas: [{
			version: [0, 0]
			schema: {
				rootfield: {
					branch:    1
					branchone: string
				} | {
					branch:    2
					branchtwo: string
				}
			}
		}, {
			version: [0, 1]
			schema: {
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
			}
		}]

		lenses: [{
			to: [0, 0]
			from: [0, 1]
			input: _
			result: {
				// FIXME this extra conditional is needed b/c otherwise lienage validation complains 'branch' field doesn't exist
				//				if input.rootfield.branch != _|_ {
				//					if input.rootfield.branch == 1 || input.rootfield.branch == 2 {
				//						// For branches 1 and 2, there are no changes, so just
				//						// unify the input directly as the result by embedding it
				//						input
				//					}
				//					if input.rootfield.branch == 3 {
				//						// For branch 3, there is no real correspondence, so choose
				//						// to fall back to branch 1 with a filler value.
				//						rootfield: {
				//							branch:    1
				//							branchone: "wasthree"
				//						}
				//					}
				//				}
			}
			lacunas: []
		}]
	}
}
