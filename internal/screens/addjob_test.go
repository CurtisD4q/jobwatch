package screens

import (
	"context"
	"strings"
	"testing"

	loom "github.com/CurtisD4q/mint-loom"
	"github.com/CurtisD4q/mint-tools/action"
)

// tabExec serves a crontab for crontab -l.
type tabExec struct{ tab string }

func (e tabExec) Run(_ context.Context, p action.Pipeline) action.Result {
	if p[0].Argv[0] == "crontab" && len(p[0].Argv) > 1 && p[0].Argv[1] == "-l" {
		return action.Result{Stdout: []byte(e.tab)}
	}
	return action.Result{}
}

func TestAddJobsListsTheCrontab(t *testing.T) {
	b := &loom.Book{Dir: t.TempDir(), Exec: tabExec{tab: "MAILTO=me\n30 2 * * * restic backup /data\n15 3 * * * /bin/sync.sh > /tmp/log\n45 5 * * * /usr/local/bin/jobwatch --loom-record photos -- rsync -a /p /q\n"}}
	frame := text(AddJobs(Options{Book: b, Program: "/usr/local/bin/jobwatch"}))
	for _, w := range []string{"Your crontab", "30 2 * * *  restic backup /data", "not recorded", "needs the shell", "45 5 * * *  rsync -a /p /q", "recorded as photos"} {
		if !strings.Contains(frame, w) {
			t.Errorf("lacks %q:\n%s", w, frame)
		}
	}
	if strings.Contains(frame, "MAILTO") {
		t.Error("a setting was listed as a job")
	}
	if !strings.Contains(text(AddJobs(Options{Book: &loom.Book{Dir: t.TempDir(), Exec: tabExec{}}})), "no job lines") {
		t.Error("empty crontab")
	}
	if len(Entries(Options{Book: b})) != 6 {
		t.Error("Add a job is not on the home list")
	}
}

func TestSuggestName(t *testing.T) {
	for in, want := range map[string]string{
		"restic backup /data":                    "restic-backup",
		"/usr/bin/updatedb":                      "updatedb",
		"/home/me/bin/sync.sh":                   "sync",
		"/usr/bin/borg create --stats /b::{now}": "borg-create",
		"rsync -a /p /q":                         "rsync",
		"/opt/Weird_Tool.py --x":                 "weird-tool",
	} {
		if got := suggestName(in); got != want {
			t.Errorf("suggestName(%q) = %q, want %q", in, got, want)
		}
	}
}
