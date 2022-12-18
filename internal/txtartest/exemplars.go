package txtartest

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/grafana/thema"
	"github.com/grafana/thema/exemplars"
)

var exonce sync.Once
var exmap map[string]thema.Lineage

func getExemplars(rt *thema.Runtime) map[string]thema.Lineage {
	if rt == nil || rt == Runtime() {
		exonce.Do(func() {
			exmap = exemplars.All(Runtime())
		})
	}

	ret := make(map[string]thema.Lineage)
	for k, v := range exmap {
		ret[k] = v
	}
	return ret
}

func exemplarNameFromPath(path string) string {
	name := filepath.Base(path)
	if name == "" || filepath.Ext(name) != ".txtar" || !strings.HasPrefix(name, "exemplar_") {
		return ""
	}
	return name[9 : len(name)-6]
}
