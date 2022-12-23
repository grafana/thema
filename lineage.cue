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
	joinSchema: {...}

	// The name of the thing specified by the schemas in this lineage.
	//
	// A lineage's name must not change as it evolves.
	name: string
	// TODO(must) https://github.com/cue-lang/cue/issues/943
	// name: must(isconcrete(name), "all lineages must have a name")

	// schemas is the ordered list of all schemas in the lineage. Each element is a
	// #SchemaDecl.
	schemas: [...#SchemaDecl & { _#schema: joinSchema }] & list.MinItems(1)

	schemas: L=[for i, decl in schemas {
		#SchemaDecl & {
			// schema: struct.MinFields(1) // TODO might this be a good way to enforce non-empty schema?
//			_#schema: joinSchema // unify joinSchema with all schemas in lineage
			if i != 0 {
				lens: reverse: {
					to: L[i-1]._#schema
					from: L[i]._#schema
				}
				if L[i].version[1] == 0 {
					lens: forward: {
						to: L[i]._#schema
						from: L[i-1]._#schema
					}
				}
			}
		}
	}]

	_allv: [...#SyntacticVersion] & [for _, sch in schemas { sch.version }]

	let pos = [ 0, for i, sch in schemas[1:] if schemas[i].version[0] < sch.version[0] { i+1 } ]

	// _counts tracks the number of versions in each major version in the lineage.
	// The index corresponds to the major version number, and the value is the
	// number of minor versions within that major.
	_counts: [for i, idx in pos[:len(pos)-1] {
		pos[i+1]-list.Sum(pos[:i+1])
	}, len(schemas)-pos[len(pos)-1]]
	// TODO check subsumption (backwards compat) of each schema with its successor natively in CUE
}

// #SchemaDecl represents a single schema declaration in Thema. In addition to
// the schema itself, it contains the schema's version, optional examples,
// composition instructions, and lenses that map to or from the schema, as
// required by Thema's invariants.
//
// Note that the version number must be explicitly declared, even though the
// correct value is algorithmically determined.
#SchemaDecl: {
	// version is the Syntactic Version number of the schema. While this property
	// is settable by lineage authors, it has exactly one correct value for any
	// particular #SchemaDecl in any lineage, algorithmically determined by its
	// position in the list of schemas and the number of its predecessors that
	// make breaking changes to their schemas.
	//
	// Despite there being only one correct choice, lineage authors must still
	// explicitly declare the schema version. Future improvements in Thema may make
	// this unnecessary, but explicitly declaring the version is always useful for
	// readability.
	//
	// Declaring a version number inconsistent with the actual set of breaking
	// changes results in an invalid lineage.
	version: #SyntacticVersion

	schema: {...}

	// Thema's internal handle for the user-provided schema definition. This
	// handle is used by all helpers/operations in the thema package. As a
	// CUE definition, use of this handle means that all thema schemas are
	// always recursively closed by default.
	//
	// This handle is also unified with the joinSchema of the containing lineage.
	_#schema: schema
//	_#schema: schema & _join

	// The joinSchema of the containing lineage.
//	_join: _

	// examples is an optional set of named examples of the schema, intended
	// for use in documentation or other non-functional contexts.
	examples?: [string]: _#schema

	// lens defines a bidirectional relation between this schema and its
	// predecessor. These relations describe how instances of the predecessor
	// schema are to be transformed into instances of this schema ("forward"),
	// and how instances of this schema are to be transformed into its
	// predecessor ("reverse").
	//
	// Depending on the version, the lens relation may contain zero, one, or two
	// transforms:
	//  - 0.0 - zero transforms. A lineage's first schema has no predecessor.
	//  - n.(x>0) - reverse transform only. A minor version indicates schema
	//    changes were backwards compatible, so no explicit transform is necessary.
	//  - (x>0).0 - forward and reverse transform. A breaking change requires
	//    explicit changes written in both directions.
	lens: {
		if version[1] == 0 {
			// First schema has no lens
			if version[0] == 0 {
				close({})
			}
			// First schema in non-0 major has a MajorLens, with two transforms
			if version[0] != 0 {
				#MajorLens
			}
		}
		if version[1] != 0 {
			// Schemas with non-zero minor versions have a MinorLens, with one transform
			#MinorLens
		}
	}
}

// MajorLens is a lens between schemas in different major versions - the higher-versioned schema is not compatible with
// the lower-versioned schema.
#MajorLens: {
	forward: #Transform
	reverse: #Transform
}

// MinorLens is a lens between schemas in the same major versions - the higher-versioned schema is compatible with
// the lower-versioned schema.
#MinorLens: {
	reverse: #Transform
}

// Transform defines the mapping from one schema to another schema in a lineage,
// and the lacunas that may exist for specific objects when moving between these schemas.
#Transform: {
	// The schema this transform is mapping to.
	to: {...}

	// The schema this transform is mapping from. Also where the input instance
	// is placed as an argument to the map.
	from: {...}

	// The mapping between the 'from' and 'to' schemas.
	//
	// The value must be an instance of the 'to' schema, constructed  be equivalent
	// to an instance of the 'to' schema, constructed through references to the 'from' schema.
	//
	// For example, if the 'from' and 'to' schemas are:
	//   from: { a: string }
	//   to:   { b: string }
	//
	// and the goal is to remap the field 'a' to be called 'b'. The following 'map'
	// accomplishes this goal:
	//   map: { b: from.a }
	map: {...}

	// lacunas describe semantic gaps in the transform's mapping. See lacuna docs
	// (TODO) for more information.
	lacunas?: [...#Lacuna]
}

// LatestVersion returns the SyntacticVersion of a lineage's latest schema.
//
// Take care in using this. If any code that depends on schema contents relies
// on it, that code will break as soon as a breaking schema change is made. This
// may be desirable within a tight development loop - e.g., for a finite team,
// working within a single repository - in order to force updating code that
// must be kept in sync.
//
// But using it in, for example, an API client based on Thema lineages
// undermines the entire goal of Thema, as it would forces breaking changes
// immediately on the client's users, rather than allowing them to update at
// their own pace.
//
// TODO functionize
#LatestVersion: {
	lin: #Lineage
	out: lin.schemas[len(lin.schemas)-1].version
}

// Get a single schema from a lineage.
#Pick: {
	lin: #Lineage
	// The schema version to pick. Either:
	//
	//   * An exact #SyntacticVersion: [1, 0]
	//   * Just the major version number: [1]
	//
	// The latter form will select the latest schema within the given
	// sequence.
	v: #SyntacticVersion | [uint64]
//	v: [<len(lin.seqs), <len(lin.seqs[v[0]].schemas)] | [<len(lin.seqs)]
	v: [<len(lin._counts), <=lin._counts[v[0]]] | [<=len(lin._counts)]
	// TODO(must) https://github.com/cue-lang/cue/issues/943
	// must(isconcrete(v[0]), "must specify a concrete sequence number")

	let _v = #SyntacticVersion & [
		v[0],
		if len(v) == 2 {v[1]},
		if len(v) == 1 {lin._counts[v[0]]},
	]

	out: {for sch in lin.schemas if sch.version == _v {sch._#schema}}
}

// SyntacticVersion is an ordered pair of non-negative integers. It represents
// the version of a schema within a lineage, or the version of an instance that
// is valid with respect to the schema of that version.
#SyntacticVersion: [uint64, uint64]

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
	// TODO check what happens when out of bounds
	out: {for i, sch in lin.schemas if sch.version == v {i}}
}
