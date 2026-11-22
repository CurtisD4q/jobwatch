package screens

import (
	"strings"
	"testing"
	"time"

	loom "github.com/CurtisD4q/mint-loom"
	"github.com/CurtisD4q/mint-loom/health"
	"github.com/CurtisD4q/mint-tools/run"
)

func TestSparkline(t *testing.T) {
	if got := sparkline([]float64{0, 1, 2, 3, 4, 5, 6, 7}); got != "▁▂▃▄▅▆▇█" {
		t.Fatal(got)
	}
	// nearly the same lengths look the same
	if got := sparkline([]float64{600, 610, 590}); got != "███" {
		t.Fatal(got)
	}
	if sparkline(nil) != "" {
		t.Fatal("empty")
	}
}

func TestSizeAndLength(t *testing.T) {
	for in, want := range map[int64]string{0: "", 999: "999 B", 1500: "1.5 kB", 8_200_000_000: "8.2 GB", 42_000_000_000: "42 GB"} {
		if got := size(in); got != want {
			t.Errorf("size(%d) = %q, want %q", in, got, want)
		}
	}
	for in, want := range map[time.Duration]string{0: "—", 42 * time.Second: "42s", 12*time.Minute + 20*time.Second: "12 min", 90 * time.Minute: "1.5 h"} {
		if got := length(in); got != want {
			t.Errorf("length(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestEvidenceShowsRuns(t *testing.T) {
	b := &loom.Book{Dir: t.TempDir(), Exec: run.Local{}}
	for _, argv := range [][]string{{"true"}, {"true"}, {"false"}} {
		loom.HandleArgs(b, append([]string{loom.RecordFlag, "nightly", "--"}, argv...))
	}
	frame := text(Evidence(Options{Book: b}, "nightly"))
	for _, w := range []string{"Runs", "Lengths, oldest to newest:", "failed (exit 1)", "done"} {
		if !strings.Contains(frame, w) {
			t.Errorf("lacks %q:\n%s", w, frame)
		}
	}
	runs, _ := b.History("nightly", 1)
	d := runDetail(runs[0])
	if !strings.Contains(d, "Exit 1") || !strings.Contains(d, "Run: false") {
		t.Fatalf("detail:\n%s", d)
	}
}

func TestEvidenceSettings(t *testing.T) {
	b := &loom.Book{Dir: t.TempDir(), Exec: run.Local{}}
	loom.HandleArgs(b, []string{loom.RecordFlag, "nightly", "--", "false"})
	frame := text(Evidence(Options{Book: b}, "nightly"))
	for _, w := range []string{"Settings", "Alerts about failing", "sent", "Delete its history", "cannot be undone"} {
		if !strings.Contains(frame, w) {
			t.Errorf("lacks %q:\n%s", w, frame)
		}
	}
	if err := health.SetMute(b.Dir, health.Mute{Job: "nightly", Kind: health.Failing}, time.Now()); err != nil {
		t.Fatal(err)
	}
	frame = text(Evidence(Options{Book: b}, "nightly"))
	if !strings.Contains(frame, "problem (muted)") || !strings.Contains(frame, "muted") {
		t.Fatalf("not shown as muted:\n%s", frame)
	}
}
