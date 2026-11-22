package screens

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CurtisD4q/jobwatch/internal/jw"
	loom "github.com/CurtisD4q/mint-loom"
)

func TestBackgroundCheckScreen(t *testing.T) {
	b := &loom.Book{Dir: t.TempDir(), Exec: tabExec{}}
	o := Options{Book: b, Program: "/usr/local/bin/jobwatch"}
	frame := text(BackgroundCheck(o))
	for _, w := range []string{"Check", "off", "Mode", "sends alerts", "Desktop notifications", "Shadow mode writes"} {
		if !strings.Contains(frame, w) {
			t.Errorf("lacks %q:\n%s", w, frame)
		}
	}
	// Saving the check (as turning it on does first) and choosing shadow
	// mode shows on the screen, and is what the schedule will run.
	if _, err := jw.ScheduleCheck(o.jwOpts(), "0 * * * *"); err != nil {
		t.Fatal(err)
	}
	if err := jw.SetCheckModes(o.jwOpts(), true, true); err != nil {
		t.Fatal(err)
	}
	frame = text(BackgroundCheck(o))
	for _, w := range []string{"off (was 0 * * * *)", "shadow, sends nothing", "Desktop notifications"} {
		if !strings.Contains(frame, w) {
			t.Errorf("lacks %q:\n%s", w, frame)
		}
	}
	r, err := b.Load(jw.CheckRecipe)
	if err != nil {
		t.Fatal(err)
	}
	argv := strings.Join(r.Steps[0].Run[0].Argv, " ")
	if argv != "/usr/local/bin/jobwatch check --log "+jw.AlertsPath(o.jwOpts())+" --shadow --notify" {
		t.Fatalf("the check runs %q", argv)
	}
}

func TestAlertsScreen(t *testing.T) {
	b := &loom.Book{Dir: t.TempDir(), Exec: tabExec{}}
	o := Options{Book: b}
	if f := text(Alerts(o)); !strings.Contains(f, "Nothing sent yet") || !strings.Contains(f, "Nothing from shadow checks") {
		t.Fatal(f)
	}
	dir := filepath.Join(b.Dir, "health")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "alerts.log"), []byte("2026-11-20T02:10:00Z backup: Its last run failed.\n"), 0o600)
	os.WriteFile(filepath.Join(dir, "shadow.log"), []byte("2026-11-21T02:10:00Z sync: Its slowest runs end only 4 min before prune starts at 03:00.\nnot a stamped line\n"), 0o600)
	f := text(Alerts(o))
	if !strings.Contains(f, "backup: Its last run failed.") || !strings.Contains(f, "sync: Its slowest runs") || strings.Contains(f, "not a stamped line") {
		t.Fatal(f)
	}
	if strings.Index(f, "Sent") > strings.Index(f, "backup:") || strings.Index(f, "Shadow: not sent") > strings.Index(f, "sync:") {
		t.Fatal("an alert is under the wrong heading:\n" + f)
	}
}
