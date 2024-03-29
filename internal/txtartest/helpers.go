package txtartest

import (
	"path"
	"sync"
	"testing"

	"github.com/grafana/thema"
)

// ForEachSchema iterates over the provided lineage's schemas, calling the
// provided test func for each one. The call is made from within a new
// Go subtest, named by the schema version.
func ForEachSchema(t *LineageTest, lin thema.Lineage, f func(*LineageTest, thema.Schema)) {
	t.Helper()
	var mu sync.Mutex
	for sch := lin.First(); sch != nil; sch = sch.Successor() {
		tsch := sch
		t.Run(tsch.Version().String(), func(gt *testing.T) {
			// FIXME this isn't safe for parallel tests
			mu.Lock()
			origt, origp, origA := t.T, t.prefix, t.Archive
			t.T, t.prefix = gt, path.Join(t.prefix, tsch.Version().String())
			f(t, tsch)
			t.T, t.prefix, t.Archive = origt, origp, origA
			mu.Unlock()
		})
	}
}
