package screens

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"

	loom "github.com/CurtisD4q/pmin-loom"
	"github.com/CurtisD4q/pmin-tools/action"
	"github.com/CurtisD4q/pmin-tools/run"
)

// counting serves a crontab, counts how often it is read, and runs
// everything else for real.
type counting struct {
	tab   string
	reads atomic.Int32
}

func (c *counting) Run(ctx context.Context, p action.Pipeline) action.Result {
	if p[0].Argv[0] == "crontab" && len(p[0].Argv) > 1 && p[0].Argv[1] == "-l" {
		c.reads.Add(1)
		return action.Result{Stdout: []byte(c.tab)}
	}
	return run.Local{}.Run(ctx, p)
}

// A screen is refreshed on every key. The crontab is read once, not once a
// key, and read again after jobwatch changes something.
func TestCrontabIsNotReadOnEveryKey(t *testing.T) {
	c := &counting{tab: "30 2 * * * restic backup /data\n"}
	o := Options{Book: &loom.Book{Dir: t.TempDir(), Exec: c}, snap: newSnapshot()}
	scr := AddJobs(o)
	for i := 0; i < 10; i++ {
		scr.Refresh(scr)
	}
	if n := c.reads.Load(); n != 1 {
		t.Fatalf("crontab read %d times for 10 refreshes", n)
	}
	o.changed()
	scr.Refresh(scr)
	if n := c.reads.Load(); n != 2 {
		t.Fatalf("crontab not read again after a change (%d reads)", n)
	}
}

// The check is shared, and worked out again as soon as a run is recorded.
func TestCheckFollowsTheHistory(t *testing.T) {
	b := &loom.Book{Dir: t.TempDir(), Exec: &counting{}}
	o := Options{Book: b, snap: newSnapshot()}
	loom.HandleArgs(b, []string{loom.RecordFlag, "nightly", "--", "true"})
	_, first, _, err := o.check()
	if err != nil || len(first.Statuses) != 1 || first.Statuses[0].State != "new" && first.Statuses[0].State != "fine" {
		t.Fatalf("%+v %v", first.Statuses, err)
	}
	_, again, _, _ := o.check()
	if &again.Statuses[0] != &first.Statuses[0] {
		t.Fatal("the same history was checked twice")
	}
	loom.HandleArgs(b, []string{loom.RecordFlag, "nightly", "--", "false"})
	_, after, _, _ := o.check()
	if after.Statuses[0].State != "failing" {
		t.Fatalf("a new failed run was not seen: %+v", after.Statuses[0])
	}
	if !strings.Contains(text(Jobs(o)), "failing") || !strings.Contains(text(Evidence(o, "nightly")), "failing") {
		t.Fatal("the list and the job disagree")
	}
}
