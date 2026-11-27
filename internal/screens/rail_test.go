package screens

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	mint "github.com/CurtisD4q/pmin-kit"
	loom "github.com/CurtisD4q/pmin-loom"
	"github.com/CurtisD4q/pmin-tools/run"
)

// inApp draws a page as the program would, with pmin-kit's own page rail.
func inApp(open func() *mint.Screen, cols int) string {
	app := &mint.App{Title: "Jobwatch", Items: []mint.HomeItem{{Icon: "x", Label: "T", Open: func() *mint.Screen {
		s := open()
		if s.Refresh != nil {
			s.Refresh(s)
		}
		return s
	}}}}
	return strings.Join(grid(app.FrameText(1, 80, cols), 80, cols), "\n")
}

// Wrapped text is whole beside the rail: every word is there, and no line is
// left holding a stray word.
func TestWrappedTextFitsBesideTheRail(t *testing.T) {
	for _, cols := range []int{48, 52, 80, 100, 140} {
		for _, indent := range []string{"", "   "} {
			frame := inApp(func() *mint.Screen {
				return &mint.Screen{Sections: []mint.Section{{Rows: wrapped(strings.Repeat("ab ", 300), indent)}}}
			}, cols)
			var counts []int
			total := 0
			for _, line := range strings.Split(frame, "\n") {
				if n := strings.Count(line, "ab"); n > 0 {
					counts = append(counts, n)
					total += n
				}
			}
			if total != 300 {
				t.Errorf("%d columns: %d of 300 words drawn", cols, total)
				continue
			}
			for _, n := range counts[:len(counts)-1] {
				if n != counts[0] {
					t.Errorf("%d columns, indent %d: uneven lines %v", cols, len(indent), counts)
					break
				}
			}
		}
	}
}

// The hours across the top of Day stay on one line beside the rail.
func TestDayAxisFitsBesideTheRail(t *testing.T) {
	b := &loom.Book{Dir: t.TempDir(), Exec: run.Local{}}
	base := time.Now().Add(-10 * 24 * time.Hour)
	var runs []loom.Run
	for i := 0; i < 9; i++ {
		s := base.AddDate(0, 0, i)
		runs = append(runs, loom.Run{Recipe: "photo-sync", Start: s, End: s.Add(14 * time.Minute), Elapsed: 840, OK: true, By: "import", Confidence: "low"})
	}
	if err := b.ReplaceImport("photo-sync", runs); err != nil {
		t.Fatal(err)
	}
	for _, cols := range []int{48, 52, 80, 100, 140} {
		frame := inApp(func() *mint.Screen {
			return Day(Options{Book: b, snap: newSnapshot(), Version: "test"})
		}, cols)
		found := false
		for _, line := range strings.Split(frame, "\n") {
			if strings.Contains(line, "00") && strings.Contains(line, "06") {
				found = strings.Contains(line, "18") && strings.Contains(line, "24")
				break
			}
		}
		if !found {
			t.Errorf("%d columns: the hours are not on one line\n%s", cols, frame)
		}
	}
}

// A note under a job's bar starts where the bars do.
func TestDayNoteLinesUpWithTheBars(t *testing.T) {
	b := &loom.Book{Dir: t.TempDir(), Exec: run.Local{}}
	base := time.Now().Add(-10 * 24 * time.Hour)
	// Recorded runs that took less than a second leave no bar to draw, which
	// a note says instead. They are written as the book keeps them, so none
	// can happen to cross into the next second.
	var lines strings.Builder
	for i := 0; i < 9; i++ {
		at := base.AddDate(0, 0, i).UTC().Format(time.RFC3339)
		fmt.Fprintf(&lines, `{"v":2,"id":"quick-%d","recipe":"quick","start":%q,"end":%q,"elapsed":0,"by":"recorded","ok":true}`+"\n", i, at, at)
	}
	if err := os.MkdirAll(filepath.Join(b.Dir, "runs"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(b.Dir, "runs", "quick.jsonl"), []byte(lines.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, cols := range []int{52, 80, 100, 140} {
		frame := inApp(func() *mint.Screen {
			return Day(Options{Book: b, snap: newSnapshot(), Version: "test"})
		}, cols)
		axis, note := -1, -1
		for _, line := range strings.Split(frame, "\n") {
			if i := strings.Index(line, "00 "); i >= 0 && strings.Contains(line, "06") && axis < 0 {
				axis = utf8.RuneCountInString(line[:i])
			}
			if i := strings.Index(line, "no end time recorded"); i >= 0 {
				note = utf8.RuneCountInString(line[:i])
			}
		}
		if axis < 0 || note != axis {
			t.Errorf("%d columns: the note starts at %d, the bars at %d\n%s", cols, note, axis, strings.TrimSpace(frame))
		}
	}
}
