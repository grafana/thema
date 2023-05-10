package thema

import "list"

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
					all:  42
					init: "some string"
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
				renamed:     string
				optional?:   int32
				withDefault: "foo" | *"bar" | "baz"
			}
		},
		{
			version: [1, 1]
			schema: {
				renamed:     string
				optional?:   int32
				withDefault: "foo" | *"bar" | "baz" | "bing"
			}
		},
	]

	lenses: [
		{
			to: [0, 3]
			from: [1, 0]
			input: _
			result: {
				init: input.renamed
				all:  input.all
				if (input.optional != _|_) {
					optional: input.optional
				}

				withDefault: input.withDefault
			}
		},
		{
			to: [0, 1]
			from: [0, 2]
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
			to: [0, 2]
			from: [0, 3]
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
			to: [1, 0]
			from: [0, 3]
			input: _
			result: {
				renamed: input.init
				all:     input.all
				if (input.optional != _|_) {
					optional: input.optional
				}

				withDefault: input.withDefault
			}
		},
		{
			to: [1, 0]
			from: [1, 1]
			input: _
			result: {
				renamed: input.renamed
				all:     input.all
				if (input.optional != _|_) {
					optional: input.optional
				}

				withDefault: input.withDefault
			}
		},
		{
			to: [0, 0]
			from: [0, 1]
			input: _
			result: {
				init: input.init
				all:  input.all
			}
		},
	]
	_counts: [4, 2]
	_basis: [0, 4]
}

cmpsv: {
	cases: [...{input: [...#SyntacticVersion], out: [...#SyntacticVersion]}]
	cases: [
		{
			input: [[0, 0], [1, 0], [1, 1], [3, 2], [0, 1]]
			out: [[0, 0], [0, 1], [1, 0], [1, 1], [3, 2]]
		},

	]
	results: [ for case in cases {
		case.out & list.Sort(case.input, {
			x:    #SyntacticVersion
			y:    #SyntacticVersion
			less: (_cmpSV & {l: x, r: y}).out == -1
		})
	},
	]
}
