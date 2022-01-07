package thema

import (
	"fmt"

	"cuelang.org/go/cue"
)

// Translate takes an instance, known to be valid with respect to one schema in
// a lineage, and transforms it to be valid with respect to another schema in
// that lineage. A new instance is returned representing the transformed value,
// along with any lacunae accumulated along the way.
//
// Forward translation within a sequence (e.g. [0, 0] to [0, 7]) is trivial, as
// all those schema changes are established as backwards compatible by Thema's
// lineage invariants. In such cases, the lens is referred to as implicit, as
// the lineage author does not write it, with translation relying on simple
// unification. Lacunae cannot be emitted from such translations.
//
// Forward translation across sequences (e.g. [0, 0] to [1, 0]), and all reverse
// translation regardless of sequence boundaries (e.g. [1, 2] to either [1, 0]
// or [0, 0]), is nontrivial and relies on explicitly defined lenses, which
// introduce room for lacunae, author judgment, and bugs.
//
// Translations are non-invertible over instances in the general case. That is,
// Thema does not guarantee that translating from [0, 0] to [1, 0] and back to
// [0, 0] will result in the exact original input.
//
// NOTE reverse translation s not yet supported, and attempting it will panic.
//
// TODO define this in terms of Instance.AsSuccessor/AsPredecessor, rather than
// those in terms of this.
func Translate(inst *Instance, to SyntacticVersion) (*Instance, TranslationLacunae) {
	if to.less(inst.Schema().Version()) {
		panic("TODO translation from newer to older schema is not yet implemented")
	}
	newsch, err := Pick(inst.Schema().Lineage(), to)
	if err != nil {
		panic(fmt.Sprintf("no schema in lineage with version %v, cannot translate", to))
	}

	out, err := cueArgs{
		"linst": inst.asLinkedInstance(),
		"to":    to,
	}.call("#Translate", inst.lib())
	if err != nil {
		// This can't happen without a name change or an invariant violation
		panic(err)
	}

	lac := make(multiTranslationLacunae, 0)
	out.LookupPath(cue.MakePath(cue.Str("lacunae"))).Decode(&lac)

	return &Instance{
		raw:  out.LookupPath(cue.MakePath(cue.Str("linst"), cue.Str("inst"))),
		name: inst.name,
		sch:  newsch,
	}, lac
}

type multiTranslationLacunae []struct {
	V   SyntacticVersion `json:"v"`
	Lac []Lacuna         `json:"lacunae"`
}

func (lac multiTranslationLacunae) AsList() []Lacuna {
	// FIXME This loses info, naturally - need to rework the lacuna types
	var l []Lacuna
	for _, v := range lac {
		l = append(l, v.Lac...)
	}
	return l
}

// func TranslateComposed(lin ComposedLineage) {

// }

func (i *Instance) asLinkedInstance() cue.Value {
	li, err := cueArgs{
		"inst": i.raw,
		"lin":  i.Schema().Lineage().RawValue(),
		"v":    i.Schema().Version(),
	}.make("#LinkedInstance", i.lib())
	if err != nil {
		// This can't happen without a name change or an invariant violation
		panic(err)
	}

	return li
}
