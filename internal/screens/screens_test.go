package screens

import (
	"regexp"
	"strings"
	"testing"
	"time"

	jobhealth "github.com/CurtisD4q/pmin-jobhealth"
	mint "github.com/CurtisD4q/pmin-kit"
	loom "github.com/CurtisD4q/pmin-loom"
	"github.com/CurtisD4q/pmin-tools/run"
)

var sgr = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

func text(s *mint.Screen) string {
	if s.Refresh != nil {
		s.Refresh(s)
	}
	for _, p := range s.Panes {
		if p.Refresh != nil {
			p.Refresh(p)
		}
	}
	return sgr.ReplaceAllString(s.FrameText(34, 140), " ")
}

func TestJobwatchScreens(t *testing.T) {
	b := &loom.Book{Dir: t.TempDir(), Exec: run.Local{}}
	o := Options{Book: b}
	if !strings.Contains(text(Jobs(o)), "No jobs yet") || !strings.Contains(text(Jobs(o)), "jobwatch record NAME --") {
		t.Fatal("empty list")
	}
	var runs []loom.Run
	base := time.Date(2026, 11, 1, 2, 0, 0, 0, time.UTC)
	for i := 0; i < 9; i++ {
		s := base.AddDate(0, 0, i)
		ok := i != 8 // the last one fails
		runs = append(runs, loom.Run{Recipe: "nightly-backup", Start: s, End: s.Add(10 * time.Minute),
			Elapsed: 600, OK: ok, By: "import", Confidence: "low"})
	}
	if err := b.ReplaceImport("nightly-backup", runs); err != nil {
		t.Fatal(err)
	}
	got := text(Jobs(o))
	if !strings.Contains(got, "nightly-backup ◌") || !strings.Contains(got, "lower confidence") {
		t.Fatalf("%s", got)
	}
	ev := text(Evidence(o, "nightly-backup"))
	if !strings.Contains(ev, "nightly-backup") || !strings.Contains(ev, "lower confidence") {
		t.Fatalf("%s", ev)
	}
	if !strings.Contains(ev, "nothing to report") && !strings.Contains(ev, "Nothing to report") &&
		!strings.Contains(ev, "failed") && !strings.Contains(ev, "not finished") {
		t.Fatalf("no evidence either way:\n%s", ev)
	}
	if !strings.Contains(text(Evidence(o, "no-such-job")), "No job called") {
		t.Fatal("unknown job")
	}
	// Day is drawn from the same runs. The name keeps its end, which is
	// what tells one backup from another.
	day := text(Day(o))
	if !strings.Contains(day, "backup") || !strings.Contains(day, "█") {
		t.Fatalf("%s", day)
	}
}

// Day draws each job on its own row, so choosing one opens its
// evidence from the day as from the list. The bars keep every column the
// command's picture gives them, so the two cannot say different things.
func TestDayRowsKeepThePicture(t *testing.T) {
	b := &loom.Book{Dir: t.TempDir(), Exec: run.Local{}}
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	b.Now = func() time.Time { return now }
	base := time.Date(2026, 11, 1, 2, 0, 0, 0, time.UTC)
	// Twenty runs each, so the day is drawn rather than left out.
	add := func(name string, hour, mins int) {
		var runs []loom.Run
		for i := 0; i < 20; i++ {
			s := base.AddDate(0, 0, i).Add(time.Duration(hour-2) * time.Hour)
			runs = append(runs, loom.Run{Recipe: name, Start: s,
				End: s.Add(time.Duration(mins) * time.Minute), Elapsed: float64(mins * 60),
				OK: true, By: "import", Confidence: "low"})
		}
		if err := b.ReplaceImport(name, runs); err != nil {
			t.Fatal(err)
		}
	}
	add("nightly-backup", 2, 30)
	add("sync", 3, 20)
	got := text(Day(Options{Book: b}))
	if !strings.Contains(got, "▸") || !strings.Contains(got, "sync") {
		t.Fatalf("no jobs on the day:\n%s", got)
	}
	// The two jobs' runs are drawn solid, at their own hours.
	if !strings.Contains(got, "█") {
		t.Fatalf("no drawn bar:\n%s", got)
	}
	if !strings.Contains(got, "low confidence") {
		t.Fatalf("the confidence mark is gone:\n%s", got)
	}
}

// The bar is drawn across the whole width the terminal gives it, and never
// narrower than the two columns an hour the command's picture uses.
func TestDayBarTakesTheWidth(t *testing.T) {
	cases := []struct{ cols, want int }{
		{120, 120 - dayNameWidth - 1}, // the rest of the screen, after the names
		{200, 200 - dayNameWidth - 1}, // and a wide one
		{80, 80 - dayNameWidth - 1},   // 54: a third of an hour a column
		{40, dayCols},                 // too narrow: the day wins, the names suffer
		{0, 80 - dayNameWidth - 1},    // no terminal: the 80 an unknown one gets
	}
	for _, c := range cases {
		if got := BarWidth(0, c.cols); got != c.want {
			t.Errorf("BarWidth(%d) = %d, want %d", c.cols, got, c.want)
		}
		if got := BarWidth(24, c.cols); got != c.want {
			t.Errorf("rows should not matter: BarWidth(24, %d) = %d", c.cols, got)
		}
	}
	// A bar is exactly as wide as it is asked for.
	for _, w := range []int{dayCols, 94, 174} {
		b := jobhealth.Bar{Name: "x", Start: 120, Usual: 150, Slow: 160, Next: 180, NextName: "y", Gap: 20}
		if got := len([]rune(drawBar(b, w))); got != w {
			t.Errorf("drawBar(%d) is %d wide", w, got)
		}
	}
}

// The hours are on the axis above the bars, at the width they are drawn.
func TestDayAxisFollowsTheBar(t *testing.T) {
	for _, w := range []int{dayCols, 94} {
		ax := dayAxis(w)
		if n := len([]rune(ax)); n != w {
			t.Fatalf("axis %d wide, bar %d", n, w)
		}
		for _, at := range []int{0, w / 4, w / 2, 3 * w / 4} {
			if strings.TrimSpace(ax[at:at+2]) == "" {
				t.Fatalf("no hour at %d in %q", at, ax)
			}
		}
	}
}

// A drawn bar says the same as the text one: the run solid from its start,
// the slow tail after it, the gap after that, and where the next job starts.
func TestDrawBarMarks(t *testing.T) {
	// A column is thirty minutes at 48 columns, so a two-hour run is four
	// columns of solid and the half-hour tail after it is one more.
	b := jobhealth.Bar{Name: "backup", Start: 120, Usual: 150, Slow: 165, Next: 181, NextName: "sync", Gap: 16}
	bar := drawBar(b, 48)
	for _, c := range []struct {
		col  int
		want rune
		what string
	}{
		{4, '█', "the usual run, 02:00 to 02:30"},
		{5, '▓', "the slow tail, 02:30 to 02:45"},
		{6, '┆', "where the next job starts, 03:00"},
		{0, ' ', "before the job starts"},
		{20, ' ', "the rest of the day"},
	} {
		if got := []rune(bar)[c.col]; got != c.want {
			t.Errorf("column %d should be %q for %s, is %q", c.col, c.want, c.what, got)
		}
	}

	// A gap of an hour of more shows: the space between the job's slow end
	// and the next job's start.
	gap := jobhealth.Bar{Name: "backup", Start: 120, Usual: 150, Slow: 150, Next: 480, NextName: "sync", Gap: 330}
	if g := []rune(drawBar(gap, 48)); g[7] != '─' || g[16] != '┆' {
		t.Errorf("a gap is not drawn: %q", string(g[4:18]))
	}

	// A job that runs into the next one has its tail against the mark, with
	// no gap drawn between them.
	b.Slow, b.Gap, b.Overrun = 200, 0, true
	if r := drawBar(b, 48); strings.ContainsRune(r, '─') {
		t.Errorf("a job that runs into the next one has no gap: %q", r)
	}

	// A span shorter than a column still shows: a twenty-minute tail on a
	// thirty-minute column is a cell, not nothing.
	short := drawBar(jobhealth.Bar{Name: "x", Start: 120, Usual: 150, Slow: 160}, 48)
	if []rune(short)[5] != '▓' {
		t.Errorf("a short tail vanished: %q", short)
	}

	// And a bar is exactly as wide as it is asked for.
	for _, w := range []int{dayCols, 94, 174} {
		if got := len([]rune(drawBar(b, w))); got != w {
			t.Errorf("drawBar(%d) is %d wide", w, got)
		}
	}
}

// A name keeps its end, which is what tells two jobs apart.
func TestPivot(t *testing.T) {
	if got := pivot("sync", 25); got != "sync" {
		t.Fatalf("%q", got)
	}
	long := "nightly-backup-photos-2026"
	got := pivot(long, 12)
	if n := len([]rune(got)); n != 12 {
		t.Fatalf("%q is %d wide", got, n)
	}
	if !strings.HasSuffix(got, "photos-2026") {
		t.Fatalf("%q kept the wrong end", got)
	}
}

// The picture and the words agree: a gap is said in words only where the
// picture marks where the next job starts.
func TestDayWordsMatchThePicture(t *testing.T) {
	// 02:00 for 30 min, slowest 35; sync at 03:00, which is more than 12
	// hours away in columns but under 12 hours in time.
	b := jobhealth.Bar{Name: "backup", Start: 120, Usual: 150, Slow: 155, Next: 180, NextName: "sync", Gap: 25}
	if !strings.ContainsRune(drawBar(jobhealth.Bar{Name: "x", Start: 120, Usual: 150, Slow: 155, Next: 180, NextName: "sync", Gap: 25}, 48), '┆') {
		t.Fatal("the next job's start is not marked")
	}
	if s := sayDay(b, false); s != "25 min before sync" {
		t.Fatalf("said %q", s)
	}
	// A job that never recorded how long it runs says nothing, so the row
	// keeps "open".
	if s := sayDay(jobhealth.Bar{Name: "x", Start: 120, Usual: 120, Slow: 120}, false); s != "" {
		t.Fatalf("said %q", s)
	}
	// A weekly job says which it is.
	if s := sayDay(jobhealth.Bar{Name: "w", Start: 120, Usual: 150, Slow: 155, Gap: 13 * 60, Period: weekly}, false); !strings.Contains(s, "weekly") {
		t.Fatalf("said %q", s)
	}
	// Overrunning is said as overrunning, whether or not the bar is marked.
	if s := sayDay(jobhealth.Bar{Name: "o", Start: 120, Usual: 150, Slow: 200, Next: 180, NextName: "sync", Overrun: true}, false); s != "runs past sync" {
		t.Fatalf("said %q", s)
	}
}
