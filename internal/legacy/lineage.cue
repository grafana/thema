package thema

import (
	"list"
)

// A Lineage is the top-level container in thema, holding the complete
// evolutionary history of a particular kind of object: every schema that has
// ever existed for that object, and the lenses that allow translating between
// those schema versions.
#Lineage: {
	// joinSchema governs the shape of schema that may be expressed in a
	// lineage. It is the least upper bound, or join, of the acceptable schema
	// value space; the schemas defined in this lineage must be instances of the
	// joinSchema.
	//
	// In the base case, the joinSchema is unconstrained/top - any value may be
	// used as a schema.
	//
	// A lineage's joinSchema may never change as the lineage evolves.
	//
	// TODO should it be an open struct rather than top?
	// TODO can this be a def? should it?
	joinSchema: _

	// The name of the thing being schematized in this lineage.
	name: string
	// TODO(must) https://github.com/cue-lang/cue/issues/943
	// name: must(isconcrete(name), "all lineages must have a name")

	// A Sequence is a non-empty ordered list of schemas, with the property that
	// every schema in the sequence is backwards compatible with (subsumes) its
	// predecessors.
	// #Sequence: [...joinSchema]

	// This exists because constraining with list.MinItems(1) isn't able to
	// tell the evaluator that it is always safe to reference #Sequence[0],
	// resulting in lots of garbage errors.
	//
	// Unfortunately, this allows empty lineage declarations by making the first
	// schema an actual joinSchema, which we do not want to be valid text for
	// authors to write.
	//
	// TODO figure out how to express the constraint without blowing up our Go logic
	#Sequence: [joinSchema, ...joinSchema]

	#Lens: {
		// The last schema in the previous sequence; logical predecessor
		ancestor: joinSchema
		// The first schema in this sequence; logical successor
		descendant: joinSchema
		forward: {
			to:   descendant
			from: ancestor
			rel:  descendant
			lacunas: [...#Lacuna]
			translated: to & rel
		}
		reverse: {
			to:   ancestor
			from: descendant
			rel:  ancestor
			lacunas: [...#Lacuna]
			translated: to & rel
		}
	}

	// seqs is the list of sequences of schema that comprise the overall
	// lineage, along with the lenses that allow translation back and forth
	// across sequences.
	seqs: [
		{
			schemas: #Sequence & list.MinItems(1)
		},
		...{
			schemas: #Sequence & list.MinItems(1)
			lens:    #Lens
		},
	]

	// Constrain that ancestor and descendant for each defined lens are the
	// final and initial schemas in the predecessor seq and the seq containing
	// the lens, respectively.
	//
	// FIXME figure out how to actually do this correctly
	// if len(seqs) > 1 {
	// seqs: [for seqv, seq in S {
	//     if seqv == 0 { {} }
	//     if seqv != 0 {
	//         lens: ancestor: S[seqv-1].schemas[len(S[seqv-1].schemas)-1]
	//         lens: descendant: seq.schemas[0]
	//     }
	// }]
	// }

	// TODO check subsumption (backwards compat) of each schema with its successor natively in CUE
}

_#vSch: {
	v:   #SyntacticVersion
	sch: _
}

// LatestVersion returns the SyntacticVersion of a lineage's latest schema.
//
// Take care in using this. If any code that depends on schema contents relies on it,
// that code will break as soon as a breaking schema change is made. This may be desirable
// within a tight development loop - e.g., for a finite team, working within a single
// repository - in order to force updating code that must be kept in sync.
//
// But using it in, for example, an API client based on Thema lineages
// undermines the entire goal of Thema, as it would forces breaking changes
// immediately on the client's users, rather than allowing them to update
// at their own pace.
//
// TODO functionize
#LatestVersion: {
	lin: #Lineage
	out: #SyntacticVersion & [len(lin.seqs) - 1, len(lin.seqs[len(lin.seqs)-1].schemas) - 1]
}

// Helper that flattens all schema into a single list, putting their
// version in an adjacent property.
//
// TODO functionize
_all: {
	lin: #Lineage
	out: [..._#vSch] & list.FlattenN([ for seqv, seq in lin.seqs {
		[ for schv, seqsch in seq.schemas {
			v: [seqv, schv]
			sch: seqsch
		}]
	}], 1)
}

// Helper that constructs a one-dimensional list of all the schema versions that
// exist in a lineage.
_allv: {
	lin: #Lineage
	out: [...#SyntacticVersion] & list.FlattenN(
		[ for seqv, seq in lin.seqs {
			[ for schv, _ in seq.schemas {[seqv, schv]}]
		}], 1)
}

// Get a single schema version from the lineage.
#Pick: {
	lin: #Lineage
	// The schema version to pick. Either:
	//
	//   * An exact #SyntacticVersion: [1, 0]
	//   * Just the sequence number: [1]
	//
	// The latter form will select the latest schema within the given
	// sequence.
	v: #SyntacticVersion | [int & >=0]
	v: [<len(lin.seqs), <len(lin.seqs[v[0]].schemas)] | [<len(lin.seqs)]
	// TODO(must) https://github.com/cue-lang/cue/issues/943
	// must(isconcrete(v[0]), "must specify a concrete sequence number")

	let _v = #SyntacticVersion & [
		v[0],
		if len(v) == 2 {v[1]},
		if len(v) == 1 {len(lin.seqs[v[0].schemas]) - 1},
	]

	out: lin.seqs[_v[0]].schemas[_v[1]]
	// TODO ^ apply object headers, etc.
}

// SyntacticVersion is an ordered pair of non negative integers. It represents
// the version of a schema within a lineage, or the version of an instance that
// is valid with respect to a schema having the same version.
// 
// Most version numbering systems leave it to the author to assign a version
// number. In Thema, a schema's version is a property of the position of the
// schema within the lineage's list of sequences, which in turn is governed by
// Thema's constraints on backwards compatibility and lens completeness. A
// SyntacticVersion ordered pair is a coordinate system, giving first the index
// of the sequence within the lineage, and second the index of the schema within
// that sequence.
// 
// In a Turing-incomplete language like CUE, schema/sequence backwards
// compatibility are properties that can be reliably checked by the CUE's
// evaluator. Relating version numbers to these checkable properties turns Thema
// versions into an encoding of those properties - hence the name,
// "SyntacticVersion".
#SyntacticVersion: [int & >=0, int & >=0]

// TODO functionize
_cmpSV: {
	l:   #SyntacticVersion
	r:   #SyntacticVersion
	out: -1 | 0 | 1
	out: {
		if l[0] < r[0] {-1}
		if l[0] > r[0] {1}
		if l[0] == r[0] && l[1] < r[1] {-1}
		if l[0] == r[0] && l[1] > r[1] {1}
		if l == r {0}
	}
}

// TODO functionize
_flatidx: {
	lin: #Lineage
	v:   #SyntacticVersion

	let inlin = lin

	// TODO this constraint should be fine to express, but uncommenting it seems
	// to blow up Go programs when they call in to unrelated pseudofuncs with
	// complaints about an incomplete v
	// _has: (_allv & { lin: inlin }).out & list.Contains(v)
	let all = (_all & {lin: inlin}).out
	out: {for i, sch in all if sch.v == v {i}}
}
