package screens

import (
	"context"
	"strings"
	"testing"

	loom "github.com/CurtisD4q/pmin-loom"
	"github.com/CurtisD4q/pmin-tools/action"
)

// timersExec answers systemctl list-timers, for the system and the user.
type timersExec struct{ system, user string }

func (e timersExec) Run(_ context.Context, p action.Pipeline) action.Result {
	argv := p[0].Argv
	if argv[0] != "systemctl" {
		return action.Result{}
	}
	if len(argv) > 1 && argv[1] == "--user" {
		return action.Result{Stdout: []byte(e.user)}
	}
	return action.Result{Stdout: []byte(e.system)}
}

const systemTimers = `Sun 2026-11-22 03:00:00 GMT 1h 40min left Sat 2026-11-21 03:00:02 GMT 22h ago backup.timer backup.service
Sun 2026-11-22 00:00:00 GMT 59min left  Sat 2026-11-21 00:00:01 GMT 1h ago  logrotate.timer logrotate.service
`
const userTimers = `Sun 2026-11-22 04:30:00 GMT 3h left Sat 2026-11-21 04:30:00 GMT 20h ago photos.timer photos.service
`

func TestTimerServices(t *testing.T) {
	got := timerServices(timersExec{system: systemTimers, user: userTimers})
	want := []timerService{{"backup.service", false}, {"logrotate.service", false}, {"photos.service", true}}
	if len(got) != len(want) {
		t.Fatalf("%+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%+v", got)
		}
	}
	if len(timerServices(timersExec{})) != 0 {
		t.Fatal("services from nothing")
	}
}

func TestImportScreen(t *testing.T) {
	o := Options{Book: &loom.Book{Dir: t.TempDir(), Exec: timersExec{system: systemTimers, user: userTimers}}}
	frame := text(Import(o))
	for _, w := range []string{"From the journal", "backup", "system", "photos", "yours", "A service by name", "rsync log", "cron log", "lower confidence"} {
		if !strings.Contains(frame, w) {
			t.Errorf("lacks %q:\n%s", w, frame)
		}
	}
	empty := text(Import(Options{Book: &loom.Book{Dir: t.TempDir(), Exec: timersExec{}}}))
	if !strings.Contains(empty, "No timers found here") {
		t.Fatal(empty)
	}
	if g := text(Guide(o)); !strings.Contains(g, "1. Add your jobs") || !strings.Contains(g, "It never changes a") {
		t.Fatal(g)
	}
}
