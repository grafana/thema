package thema

import (
	"struct"
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
	// All Thema schemas must be struct-kinded. As such,
	//
	// In the base case, the joinSchema is unconstrained/top - any value may be
	// used as a schema.
	//
	// A lineage's joinSchema must never change as the lineage evolves.
	joinSchema: struct.MinFields(0)

	// The name of the thing specified by the schemas in this lineage.
	//
	// A lineage's name must not change as it evolves.
	name: string
	// TODO(must) https://github.com/cue-lang/cue/issues/943
	// name: must(isconcrete(name), "all lineages must have a name")

	// The lineage-local handle for #SchemaDef, into which we have injected this
	// lineage's joinSchema.
	//	let Schema = #SchemaDef & {_join: joinSchema}
	let Schema = #SchemaDef

	// schemas is the ordered list of all schemas in the lineage. Each element is a
	// #SchemaDef.
	schemas: [...]

	schemas: S=[ for i, _ in schemas {
		let cur = S[i]
		Schema & {
			examples: [string]: cur._#schema
			//			_#schema: cur.schema & joinSchema
			if i != 0 {
				let pre = S[i-1]
				lenses: {
					input: {
						self:  _
						prior: _
						// TODO(must) https://github.com/cue-lang/cue/issues/943
						_selfMust:  self & cur._#schema
						_priorMust: prior & pre._#schema
					}
				}
			}
		}
	}]

	//	counts: _counts
	// TODO check subsumption (backwards compat) of each schema with its successor natively in CUE

	// _counts tracks the number of versions in each major version in the lineage.
	// The index corresponds to the major version number, and the value is the
	// number of minor versions within that major.
	_counts: [...uint64]
	if len(schemas) == 1 {
		_counts: [0]
	}

	if len(schemas) > 1 {
		let pos = [0, for i, sch in list.Drop(schemas, 1) if schemas[i].version[0] < sch.version[0] {i + 1}]
		_counts: [ for i, idx in list.Slice(pos, 0, len(pos)-1) {
			pos[i+1] - list.Sum(list.Slice(pos, 0, i+1))
		}, len(schemas) - pos[len(pos)-1]]

		// The following approach to the above:
		//
		//		let pos = [0, for i, sch in schemas[1:] if schemas[i].version[0] < sch.version[0] { i+1 }]
		//		_counts: [for i, idx in pos[:len(pos)-1] {
		//			pos[i+1]-list.Sum(pos[:i+1])
		//		}, len(schemas)-pos[len(pos)-1]]
		//
		// causes the following cue internals panic:
		// panic: getNodeContext: nodeContext out of sync [recovered]
		//	panic: getNodeContext: nodeContext out of sync
	}
}

// #SchemaDef represents a single schema declaration in Thema. In addition to
// the schema itself, it contains the schema's version, optional examples,
// composition instructions, and lenses that map to or from the schema, as
// required by Thema's invariants.
//
// Note that the version number must be explicitly declared, even though the
// correct value is algorithmically determined.
#SchemaDef: {
	// version is the Syntactic Version number of the schema. While this property
	// is settable by lineage authors, it has exactly one correct value for any
	// particular #SchemaDef in any lineage, algorithmically determined by its
	// position in the list of schemas and the number of its predecessors that
	// make breaking changes to their schemas.
	//
	// Despite there being only one correct choice, lineage authors must still
	// explicitly declare the schema version. Future improvements in Thema may make
	// this unnecessary, but explicitly declaring the version is always useful for
	// readability.
	//
	// The entire lineage is considered invalid if the version number in this field
	// is inconsistent with the algorithmically determined set of [non-]breaking changes.
	version: #SyntacticVersion

	schema: _

	// Thema's internal handle for the user-provided schema definition. This
	// handle is used by all helpers/operations in the thema package. As a
	// CUE definition, use of this handle entails that all thema schemas are
	// always recursively closed by default.
	//
	// This handle is also unified with the joinSchema of the containing lineage.
	_#schema: struct.MinFields(1) & _join & schema

	_join: struct.MinFields(0)

	// examples is an optional set of named examples of the schema, intended
	// for use in documentation or other non-functional contexts.
	examples?: [string]: _

	// lenses contains the mappings that define how to translate an instance of
	// this schema to back and forth between this schema and its
	// predecessor in the lineage.
	//
	// Within this field, there may exist a priorToSelf lens, which
	//
	// that take an instance of this schema and transform
	// them into an instance of the predecessor schema, and vice-versa.
	//
	// Within lens definitions, this schema and instances of it are referred to as
	// "self", and the predecessor schema and instances of it are referred to as "prior".
	//
	// Depending on the version, there may be zero, one, or two lenses:
	//  - 0.0 - zero lenses. A lineage's first schema has no predecessor.
	//  - n.(x>0) - selfToPrior lens only. A minor version increase entails schema
	//    changes that were backwards compatible, so no explicit lens is necessary.
	//  - (x>0).0 - selfToPrior and priorToSelf lenses. Breaking changes in the schema
	//    require lineage authors to explicitly write lenses in both directions.
	lenses: {
		// Inputs to the transforms defined within lenses. These fields may not be directly populated by lineage authors. (TODO verify)
		// Rather, values are dynamically injected by Thema as part of the translate operation.
		input: {
			// If populated, a valid instance of this schema.
			self: _#schema
			// If populated, a valid instance of the previous schema in this lineage.
			prior: _
		}

		if version[1] == 0 {
			// First schema has no lens
			if version[0] == 0 {
				close({})
			}

			// First schema in non-0 major has the MajorLenses set, with two transforms
			if version[0] != 0 {
				#MajorLenses
			}
		}
		if version[1] != 0 {
			// Schemas with non-zero minor versions have the MinorLenses set
			#MinorLenses
		}
	}
}

// MajorLenses is a lens between schemas in different major versions - the higher-versioned schema is not compatible with
// the lower-versioned schema.
#MajorLenses: {
	priorToSelf: #Lens
	selfToPrior: #Lens
}

// MinorLenses is a lens between schemas in the same major versions - the higher-versioned schema is compatible with
// the lower-versioned schema.
#MinorLenses: {
	selfToPrior: #Lens
}

// Lens defines the a transformation that maps the fields of one schema to the fields of
// another schema, as well as the lacunas that may exist for specific objects when moving
// between these schemas.
#Lens: {
	// The mapping between the 'from' and 'to' schemas.
	//
	// The value must be an instance of the 'to' schema, constructed through
	// references to the 'from' schema.
	//
	// For example, if the 'from' and 'to' schemas are:
	//   from: { a: string }
	//   to:   { b: string }
	//
	// and the goal is to remap the field 'a' to be called 'b'. The following 'map'
	// accomplishes this goal:
	//   map: { b: from.a }
	result: struct.MinFields(0)

	// lacunas describe semantic gaps in the transform's mapping. See lacuna docs
	// for more information (TODO).
	lacunas?: [...#Lacuna]
}

// LatestVersion is a pseudofunction that returns the SyntacticVersion of a lineage's
// latest schema.
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

// Pick is a pseudofunction that returns the schema from a provided Lineage
// (lin) that corresponds to the provided SyntacticVersion (v). Bounds
// constraints enforce that the provided version number exists within the
// provided lineage.
//
// Pick is the only correct mechanism to retrieve a lineage's declared schema.
// Retrieving a lineage's schemas by direct indexing will not check invariants,
// apply compositions or joinSchemas.
#Pick: {
	// The lineage from which to retrieve a schema
	lin: #Lineage

	// The schema version to retrieve. Either:
	v: #SyntacticVersion & [<len(lin._counts), <=lin._counts[v[0]]]
	// TODO(must) https://github.com/cue-lang/cue/issues/943
	// must(isconcrete(v[0]), "must specify a concrete major version")

	out: {for sch in lin.schemas if sch.version == v {sch._#schema}}
}

// PickDef takes the same arguments as Pick, but returns the entire
// #SchemaDef rather than only the schema body itself.
#PickDef: {
	// The lineage from which to retrieve a schema.
	lin: #Lineage

	// The schema version to retrieve. Either:
	v: #SyntacticVersion & [<len(lin._counts), <=lin._counts[v[0]]]
	// TODO(must) https://github.com/cue-lang/cue/issues/943
	// must(isconcrete(v[0]), "must specify a concrete sequence number")

	out: {for sch in lin.schemas if sch.version == v {sch}}
}

// SyntacticVersion is an ordered pair of non-negative integers. It represents
// the version of a schema within a lineage, or the version of an instance that
// is valid with respect to the schema of that version.
#SyntacticVersion: [uint64, uint64]

// TODO functionize
_cmpSV: FN={
	l:   #SyntacticVersion
	r:   #SyntacticVersion
	out: -1 | 0 | 1
	out: {
		if FN.l[0] < FN.r[0] {-1}
		if FN.l[0] > FN.r[0] {1}
		if FN.l[0] == FN.r[0] && FN.l[1] < FN.r[1] {-1}
		if FN.l[0] == FN.r[0] && FN.l[1] > FN.r[1] {1}
		if FN.l == FN.r {0}
	}
}

// TODO functionize
_flatidx: L={
	lin: #Lineage
	v:   #SyntacticVersion
	// TODO check what happens when out of bounds
	out: {for i, sch in L.lin.schemas if sch.version == v {i}}
}
