package thema

basic: #Lineage & {
	name: "basic"
	joinSchema: {
		all: int32
	}
	schemas: [
		{
			version: [0, 0]
			schema: {
				init: string
			}
			examples: {
				simple: {
					init: "foo"
				}
			}
		},
		{
			version: [0, 1]
			schema: {
				init:      string
				optional?: int32
			}
		},
		{
			version: [0, 2]
			schema: {
				init:        string
				optional?:   int32
				withDefault: *"foo" | "bar"
			}
		},
		{
			version: [0, 3]
			schema: {
				init:        string
				optional?:   int32
				withDefault: *"foo" | "bar" | "baz"
			}
		},
		{
			version: [1, 0]
			schema: {
				init:        string
				optional?:   int32
				withDefault: "foo" | *"bar" | "baz"
			}
		},
	]

	lenses: [
		{
			from: [0, 1]
			to: [0, 0]
			input: _
			result: {
				init: input.init
				all:  input.all
			}
		},
		{
			from: [0, 2]
			to: [0, 1]
			input: _
			result: {
				init: input.init
				all:  input.all
				if (input.optional != _|_) {
					optional: input.optional
				}
			}
		},
		{
			from: [0, 3]
			to: [0, 2]
			input: _
			result: {
				init: input.init
				all:  input.all
				if (input.optional != _|_) {
					optional: input.optional
				}

				withDefault: input.withDefault // TODO does this actually work
			}
		},
		{
			from: [1, 0]
			to: [0, 3]
			input: _
			result: {
				init: input.init
				all:  input.all
				if (input.optional != _|_) {
					optional: input.optional
				}

				withDefault: input.withDefault
			}
		},
	]
	_counts: [4, 1]
	_basis: [0, 4]
}
