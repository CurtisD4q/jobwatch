package screens

import (
	"testing"

	loom "github.com/CurtisD4q/mint-loom"
	"github.com/CurtisD4q/mint-tools/run"
)

// A screen asks to be drawn again when a run is recorded, and not when
// nothing changed.
func TestLiveSeesNewRuns(t *testing.T) {
	b := &loom.Book{Dir: t.TempDir(), Exec: run.Local{}}
	o := Options{Book: b, snap: newSnapshot()}
	poll := o.live()
	for i := 0; i < 5; i++ {
		if poll(nil) {
			t.Fatal("redrawn with nothing changed")
		}
	}
	loom.HandleArgs(b, []string{loom.RecordFlag, "nightly", "--", "true"})
	if !poll(nil) {
		t.Fatal("a new run was not noticed")
	}
	if poll(nil) {
		t.Fatal("the same change was noticed twice")
	}
}
