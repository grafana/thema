package thema

import (
	"fmt"

	"cuelang.org/go/cue"
)

// Translate takes an instance
func Translate(inst *Instance, to SyntacticVersion) (*Instance, TranslationLacunae) {
	if to.less(inst.Schema().Version()) {
		panic("TODO translation from newer to older schema is not yet implemented")
	}
	newsch, err := Pick(inst.Schema().Lineage(), to)
	if err != nil {
		fmt.Println(err)
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
