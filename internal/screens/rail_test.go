package screens

import (
	"strings"
	"testing"
	"time"

	mint "github.com/CurtisD4q/pmin-kit"
	loom "github.com/CurtisD4q/pmin-loom"
	"github.com/CurtisD4q/pmin-tools/run"
)

// besideRail is the columns a page gets in an App on a terminal this wide:
// pmin-kit's page rail takes four on a screen of 48 columns or more. A test
// cannot give a screen a rail, so it draws the page that much narrower.
func besideRail(cols int) int {
	if cols >= 48 {
		return cols - 4
	}
	return cols
}

// Wrapped text fits beside the rail: no row is wrapped a second time, which
// would leave a word or two on a line of their own.
func TestWrappedTextFitsBesideTheRail(t *testing.T) {
	old := termCols
	t.Cleanup(func() { termCols = old })
	for _, cols := range []int{52, 80, 100, 140} {
		cols := cols
		termCols = func() int { return cols }
		for _, indent := range []string{"", "   "} {
			rows := wrapped(strings.Repeat("ab ", 300), indent)
			scr := &mint.Screen{Title: "T", Sections: []mint.Section{{Rows: rows}}}
			width := besideRail(cols)
			lines := 0
			for _, line := range grid(scr.FrameText(80, width), 80, width) {
				if strings.Contains(line, "ab") {
					lines++
				}
			}
			if lines != len(rows) {
				t.Errorf("%d columns, indent %d: %d rows drawn on %d lines", cols, len(indent), len(rows), lines)
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
	old := termCols
	t.Cleanup(func() { termCols = old })
	for _, cols := range []int{52, 80, 100, 140} {
		cols := cols
		termCols = func() int { return cols }
		scr := Day(Options{Book: b, snap: newSnapshot(), Version: "test"})
		scr.Refresh(scr)
		width := besideRail(cols)
		frame := strings.Join(grid(scr.FrameText(40, width), 40, width), "\n")
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
