package screens

import (
	"strings"
	"testing"

	loom "github.com/CurtisD4q/mint-loom"
)

func keysOf(t *testing.T, keys []string) map[string]bool {
	t.Helper()
	m := map[string]bool{}
	for _, k := range keys {
		m[k] = true
	}
	return m
}

// Every screen has the same keys, except the one that would open itself,
// and none of them is a key Mint or the vim style keeps.
func TestEveryScreenHasTheKeys(t *testing.T) {
	o := Options{Book: &loom.Book{Dir: t.TempDir(), Exec: tabExec{}}, snap: newSnapshot(), Version: "9.9"}
	for _, item := range Entries(o) {
		scr := item.Open()
		var ks []string
		for _, k := range scr.Keys {
			ks = append(ks, k.Key)
			if strings.Contains("bqyHL[]| jkhlgG", k.Key) {
				t.Errorf("%s takes %q, which Mint keeps", scr.Title, k.Key)
			}
		}
		got := keysOf(t, ks)
		if !got["r"] || !got["?"] && scr.Title != "How this works" || !got["o"] && scr.Title != "Overview" {
			t.Errorf("%s keys: %v", scr.Title, ks)
		}
		if scr.Title == "The day" && got["d"] {
			t.Error("the day opens itself")
		}
	}
	frame := text(Settings(o))
	for _, w := range []string{"d   The day", "r   read the history", "History", "Key style", "default", "Version", "9.9"} {
		if !strings.Contains(frame, w) {
			t.Errorf("settings lack %q:\n%s", w, frame)
		}
	}
}
