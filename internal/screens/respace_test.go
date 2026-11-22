package screens

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	loom "github.com/CurtisD4q/mint-loom"
)

func TestShiftedLine(t *testing.T) {
	cl := loom.CronLine{Schedule: "30 2 * * *", Command: "/usr/bin/jobwatch --loom-record b -- restic backup /d"}
	if got, ok := shiftedLine(cl, 45); !ok || got != "15 3 * * * /usr/bin/jobwatch --loom-record b -- restic backup /d" {
		t.Fatalf("%q %v", got, ok)
	}
	if got, ok := shiftedLine(cl, -150); !ok || !strings.HasPrefix(got, "0 0 * * *") {
		t.Fatalf("%q %v", got, ok)
	}
	if _, ok := shiftedLine(cl, -151); ok {
		t.Fatal("a move into yesterday was made")
	}
	for _, sched := range []string{"*/5 * * * *", "0 2-4 * * *", "@daily"} {
		if _, ok := shiftedLine(loom.CronLine{Schedule: sched, Command: "x"}, 10); ok {
			t.Errorf("%q was shifted", sched)
		}
	}
	if d := minutesBetween("23:50", "00:20"); d != 30 {
		t.Fatal(d)
	}
	if d := minutesBetween("02:00", "01:00"); d != -60 {
		t.Fatal(d)
	}
}

func writeRuns(t *testing.T, dir, name string, runs []loom.Run) {
	t.Helper()
	os.MkdirAll(filepath.Join(dir, "runs"), 0o700)
	var b strings.Builder
	for _, r := range runs {
		j, _ := json.Marshal(r)
		b.Write(j)
		b.WriteString("\n")
	}
	if err := os.WriteFile(filepath.Join(dir, "runs", name+".jsonl"), []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRespaceScreen(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	for _, name := range []string{"backup", "sync", "prune"} {
		var runs []loom.Run
		for d := 1; d <= 20; d++ {
			s := now.AddDate(0, 0, -d).Truncate(24 * time.Hour).Add(2 * time.Hour)
			runs = append(runs, loom.Run{Recipe: name, Start: s, End: s.Add(40 * time.Minute), Elapsed: 2400, OK: true, By: "recorded"})
		}
		writeRuns(t, dir, name, runs)
	}
	b := &loom.Book{Dir: dir, Exec: tabExec{}, Now: func() time.Time { return now }}
	frame := text(Respace(Options{Book: b}))
	for _, w := range []string{"Move a start by up to", "2 h", "Suggested moves", "→", "move by hand", "Overlap now:", "Nothing is changed"} {
		if !strings.Contains(frame, w) {
			t.Errorf("lacks %q:\n%s", w, frame)
		}
	}
}
