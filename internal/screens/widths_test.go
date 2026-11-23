package screens

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	mint "github.com/CurtisD4q/pmin-kit"
	loom "github.com/CurtisD4q/pmin-loom"
	"github.com/CurtisD4q/pmin-tools/run"
)

// Every screen on the home list, drawn at a phone-narrow 52 columns and at
// 100, shows its title, raises no error, and cuts off no line of text. A
// path may be cut, since it can be copied whole with y.
func TestEveryScreenFitsItsWidth(t *testing.T) {
	b := &loom.Book{Dir: t.TempDir(), Exec: run.Local{}}
	for _, c := range []string{"true", "true", "false"} {
		loom.HandleArgs(b, []string{loom.RecordFlag, "nightly-backup", "--", c})
	}
	base := time.Now().Add(-10 * 24 * time.Hour)
	var imported []loom.Run
	for i := 0; i < 9; i++ {
		s := base.AddDate(0, 0, i)
		imported = append(imported, loom.Run{Recipe: "photo-sync", Start: s, End: s.Add(14 * time.Minute), Elapsed: 840, OK: true, By: "import", Confidence: "low"})
	}
	if err := b.ReplaceImport("photo-sync", imported); err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Join(b.Dir, "health"), 0o700)
	stamp := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	os.WriteFile(filepath.Join(b.Dir, "health", "shadow.log"), []byte(stamp+" photo-sync: It has not finished successfully in 3 days (it usually runs every 24 hours).\n"), 0o600)
	os.WriteFile(filepath.Join(b.Dir, "health", "alerts.log"), []byte(stamp+" nightly-backup: Its last run failed, and the run before it was slower than any in the last month.\n"), 0o600)
	o := Options{Book: b, snap: newSnapshot(), Version: "test"}
	old := termCols
	t.Cleanup(func() { termCols = old })
	for _, width := range []int{52, 100} {
		width := width
		termCols = func() int { return width }
		screens := Entries(o)
		for _, job := range []string{"nightly-backup", "photo-sync"} {
			job := job
			screens = append(screens, mint.HomeItem{Label: "job " + job, Open: func() *mint.Screen { return Evidence(o, job) }})
		}
		for _, item := range screens {
			scr := item.Open()
			if scr.Refresh != nil {
				scr.Refresh(scr)
			}
			for _, p := range scr.Panes {
				if p.Refresh != nil {
					p.Refresh(p)
				}
			}
			frame := strings.Join(grid(scr.FrameText(40, width), 40, width), "\n")
			if os.Getenv("SHOW") != "" && (item.Label == "Overview" || item.Label == "The day") && width == 52 {
				t.Log("\n" + frame)
			}
			if !strings.Contains(frame, scr.Title) {
				t.Errorf("%s @%d: no title\n%s", item.Label, width, frame)
			}
			if strings.Contains(frame, "went wrong") || strings.Contains(frame, "panic") {
				t.Errorf("%s @%d: an error\n%s", item.Label, width, frame)
			}
			for _, line := range strings.Split(frame, "\n") {
				body := strings.TrimRight(strings.Trim(line, "│ "), " ")
				if strings.HasSuffix(body, "…") && !strings.Contains(body, "/") {
					t.Errorf("%s @%d: cut off: %q", item.Label, width, body)
				}
			}
		}
	}
}

var csi = regexp.MustCompile(`^\x1b\[([0-9;?]*)([A-Za-z])`)

// grid replays a frame, as a terminal would, into h lines of w cells: the
// frame moves the cursor rather than writing new lines.
func grid(frame string, h, w int) []string {
	cells := make([][]rune, h)
	for i := range cells {
		cells[i] = []rune(strings.Repeat(" ", w))
	}
	row, col := 0, 0
	for len(frame) > 0 {
		if m := csi.FindStringSubmatch(frame); m != nil {
			if m[2] == "H" {
				r, c := 1, 1
				fmt.Sscanf(m[1], "%d;%d", &r, &c)
				row, col = r-1, c-1
			}
			frame = frame[len(m[0]):]
			continue
		}
		r, size := utf8.DecodeRuneInString(frame)
		frame = frame[size:]
		if r == '\n' {
			row, col = row+1, 0
			continue
		}
		if row >= 0 && row < h && col >= 0 && col < w {
			cells[row][col] = r
		}
		col++
	}
	out := make([]string, h)
	for i, c := range cells {
		out[i] = string(c)
	}
	return out
}
