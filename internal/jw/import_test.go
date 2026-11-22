package jw

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CurtisD4q/mint-tools/action"

	loom "github.com/CurtisD4q/mint-loom"
)

const journalText = `2026-01-06T03:00:01+0000 host systemd[1]: Started backup.service - Backup.
2026-01-06T03:00:04+0000 host restic[9]: Transferred: 1.5 GiB
2026-01-06T03:10:01+0000 host systemd[1]: backup.service: Deactivated successfully.
2026-01-06T04:00:01+0000 host systemd[1]: Started backup.service - Backup.
2026-01-06T04:01:00+0000 host systemd[1]: backup.service: Main process exited, code=exited, status=1/FAILURE
2026-01-06T04:01:00+0000 host systemd[1]: backup.service: Failed with result 'exit-code'.
`

func TestParseJournalKeepsNoMessage(t *testing.T) {
	runs := ParseJournal(journalText, "backup")
	if len(runs) != 2 {
		t.Fatalf("%d runs", len(runs))
	}
	if !runs[0].OK || runs[0].Elapsed < 500 || runs[0].Bytes == 0 {
		t.Fatalf("success %+v", runs[0])
	}
	if runs[1].OK || runs[1].Start.Hour() != 4 {
		t.Fatalf("failure %+v", runs[1])
	}
	for _, r := range runs {
		if len(r.Steps) != 0 {
			t.Fatal("stored a message")
		}
	}
	dir := t.TempDir()
	b := &loom.Book{Dir: dir}
	if err := b.ReplaceImport("backup", runs); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(dir + "/runs/imported/backup.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "Started backup") || strings.Contains(string(raw), "restic") {
		t.Fatalf("the message was stored:\n%s", raw)
	}
	if !strings.Contains(string(raw), `"confidence":"low"`) {
		t.Fatalf("%s", raw)
	}
	got, err := b.History("backup", 0)
	if err != nil || len(got) != 2 || got[0].Confidence != "low" {
		t.Fatalf("%v %+v", err, got)
	}
}

func TestImportJournalUsesJournalctl(t *testing.T) {
	f := &scripted{out: journalText}
	dir := t.TempDir()
	o := Options{Dir: dir, Exec: f}
	n, err := ImportJournal(o, "backup.service", "90d")
	if err != nil || n != 2 {
		t.Fatal(err, n)
	}
	if len(f.argv) == 0 || f.argv[0] != "journalctl" || !contains(f.argv, "backup.service") || !contains(f.argv, "--since=-90d") {
		t.Fatalf("%v", f.argv)
	}
}

func TestParseRcloneRsyncCron(t *testing.T) {
	rc := ParseRclone(`2026/01/06 03:00:00 INFO  : Local file system: starting
Transferred: 12 MiB / 12 MiB, 100%
Elapsed time: 10m0s
`, "nightly")
	if len(rc) != 1 || !rc[0].OK || rc[0].Elapsed != 600 || rc[0].Bytes != 12<<20 || len(rc[0].Steps) != 0 {
		t.Fatalf("%+v", rc)
	}
	rs := ParseRsync(`2026/01/06 03:00:00 [1] building file list
2026/01/06 03:04:00 [1] sent 1,048,576 bytes  received 128 bytes
`, "nightly")
	if len(rs) != 1 || rs[0].Elapsed != 240 || rs[0].Bytes != 1048576 {
		t.Fatalf("%+v", rs)
	}
	now := time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)
	cr := ParseCron("Jan  6 03:00:01 host CRON[10]: (user) CMD (backup.sh)\n", "nightly", now)
	if len(cr) != 1 || !cr[0].End.IsZero() || cr[0].Elapsed != 0 || cr[0].Start.Day() != 6 {
		t.Fatalf("%+v", cr)
	}
	dir := t.TempDir()
	if err := (&loom.Book{Dir: dir}).ReplaceImport("nightly", cr); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(dir + "/runs/imported/nightly.jsonl")
	if strings.Contains(string(raw), "backup.sh") {
		t.Fatalf("kept the command:\n%s", raw)
	}
}

func contains(argv []string, s string) bool {
	for _, a := range argv {
		if a == s {
			return true
		}
	}
	return false
}

type scripted struct {
	argv []string
	out  string
}

func (s *scripted) Run(_ context.Context, p action.Pipeline) action.Result {
	if len(p) > 0 {
		s.argv = p[0].Argv
	}
	return action.Result{Stdout: []byte(s.out)}
}

// journalctl refuses "90d" ("Failed to parse timestamp"); how far back to
// go is "-90d". Dates and words pass on as they are.
func TestJournalSince(t *testing.T) {
	for in, want := range map[string]string{
		"": "-90d", "90d": "-90d", "2w": "-2w", "36h": "-36h", "-7d": "-7d",
		"yesterday": "yesterday", "2026-01-01": "2026-01-01", "2026-01-01 03:00": "2026-01-01 03:00",
	} {
		if got := journalSince(in); got != want {
			t.Errorf("journalSince(%q) = %q, want %q", in, got, want)
		}
	}
}

// A user timer's service is in the person's own journal.
func TestImportJournalFromUser(t *testing.T) {
	f := &scripted{out: journalText}
	o := Options{Dir: t.TempDir(), Exec: f}
	if _, err := ImportJournalFrom(o, "backup.service", "", true); err != nil {
		t.Fatal(err)
	}
	if len(f.argv) < 2 || f.argv[1] != "--user" {
		t.Fatalf("%v", f.argv)
	}
}
