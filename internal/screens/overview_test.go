package screens

import (
	"strings"
	"testing"
	"time"

	mint "github.com/CurtisD4q/mint-kit"
	loom "github.com/CurtisD4q/mint-loom"
	"github.com/CurtisD4q/mint-loom/health"
	"github.com/CurtisD4q/mint-tools/run"
)

func overviewText(o Options) string {
	scr := Overview(o)
	return text(scr)
}

func TestOverviewFirstRun(t *testing.T) {
	o := Options{Book: &loom.Book{Dir: t.TempDir(), Exec: tabExec{}}, snap: newSnapshot()}
	frame := overviewText(o)
	for _, w := range []string{"No jobs yet", "1. Add a job from your crontab", "3. Turn on the check, in shadow", "2. Import history", "How this works", "jobwatch record NAME --", "Today", "Background check"} {
		if !strings.Contains(frame, w) {
			t.Errorf("lacks %q:\n%s", w, frame)
		}
	}
	if b := o.badges(); len(b) != 2 || b[0].Text != "no jobs yet" || b[1].Text != "check off" {
		t.Fatalf("%+v", b)
	}
}

func TestOverviewWithJobs(t *testing.T) {
	b := &loom.Book{Dir: t.TempDir(), Exec: run.Local{}}
	for _, c := range []string{"true", "true", "false"} {
		loom.HandleArgs(b, []string{loom.RecordFlag, "backup", "--", c})
	}
	loom.HandleArgs(b, []string{loom.RecordFlag, "prune", "--", "true"})
	o := Options{Book: b, snap: newSnapshot()}
	frame := overviewText(o)
	for _, w := range []string{"1 problem: backup is failing", "backup", "failing", "prune", "+ Add a job", "Latest alerts", "No alerts yet"} {
		if !strings.Contains(frame, w) {
			t.Errorf("lacks %q:\n%s", w, frame)
		}
	}
	// the worst job is listed first
	if strings.Index(frame, "backup") > strings.Index(frame, "prune") {
		t.Errorf("backup is not first:\n%s", frame)
	}
	badges := o.badges()
	if badges[0].Text != "1 problem" || badges[0].Kind != mint.BadgeError {
		t.Fatalf("%+v", badges)
	}
}

func TestOverviewBanner(t *testing.T) {
	old := termCols
	t.Cleanup(func() { termCols = old })
	termCols = func() int { return 140 }
	fine := health.Report{Statuses: []health.Status{{Job: "a", State: "fine"}}}
	if got := overviewBanner(fine, false, 1); !strings.Contains(got, "background check is off") {
		t.Fatal(got)
	}
	if got := overviewBanner(fine, true, 1); got != "" {
		t.Fatal(got)
	}
	two := health.Report{Statuses: []health.Status{{Job: "a", State: "failing", Level: health.Problem}, {Job: "b", State: "overdue", Level: health.Problem}}}
	if got := overviewBanner(two, true, 2); got != "2 problems: a is failing, b is overdue" {
		t.Fatal(got)
	}
	if got := overviewBanner(health.Report{}, false, 0); got != "" {
		t.Fatalf("no jobs, and nagged about the check: %q", got)
	}
}

func TestLocalStartRounds(t *testing.T) {
	now := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	got := localStart(179.99, now)
	want := time.Date(2026, 1, 10, 3, 0, 0, 0, time.UTC).Local().Format("15:04")
	if got != want {
		t.Fatalf("%s, want %s", got, want)
	}
}

// On a narrow terminal the banner says it more briefly, rather than being
// cut off.
func TestOverviewBannerFits(t *testing.T) {
	old := termCols
	t.Cleanup(func() { termCols = old })
	termCols = func() int { return 52 }
	many := health.Report{}
	for _, n := range []string{"nightly-backup", "photo-sync", "offsite-copy"} {
		many.Statuses = append(many.Statuses, health.Status{Job: n, State: "overdue", Level: health.Problem})
	}
	got := overviewBanner(many, true, 3)
	if len([]rune(got)) > 46 || !strings.HasPrefix(got, "3 problems: nightly-backup") {
		t.Fatalf("%q (%d)", got, len([]rune(got)))
	}
}
