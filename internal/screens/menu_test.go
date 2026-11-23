package screens

import (
	"strings"
	"testing"

	loom "github.com/CurtisD4q/pmin-loom"
)

// The menu's keys are the keys that work: each one it shows opens that
// screen, and every key jobwatch has is on it.
func TestMenuKeysAreTheRealKeys(t *testing.T) {
	onMenu := map[string]string{}
	for _, g := range menuGroups() {
		for _, it := range g.items {
			if it.key != "" {
				onMenu[it.key] = it.name
			}
		}
	}
	for _, j := range jumps() {
		if j.what == "Menu" {
			continue
		}
		if onMenu[j.key] != j.what {
			t.Errorf("%s opens %s, but the menu says %q", j.key, j.what, onMenu[j.key])
		}
		delete(onMenu, j.key)
	}
	for k, name := range onMenu {
		t.Errorf("the menu shows %s for %s, which is not a key", k, name)
	}
}

// Before there are any jobs, the menu says where to start.
func TestMenuFirstRun(t *testing.T) {
	o := Options{Book: &loom.Book{Dir: t.TempDir(), Exec: tabExec{}}, snap: newSnapshot()}
	scr := Menu(o)
	scr.Refresh(scr)
	if !strings.Contains(scr.Banner, "New here") {
		t.Errorf("banner: %q", scr.Banner)
	}
	if !strings.Contains(text(Menu(o)), "none yet") {
		t.Error("no jobs not said")
	}
}
