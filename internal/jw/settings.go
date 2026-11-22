package jw

import (
	"bufio"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	loom "github.com/CurtisD4q/mint-loom"
	"github.com/CurtisD4q/mint-tools/action"
)

// CheckSettings is how the background check is set up.
type CheckSettings struct {
	// Saved is true once the check has been set up at all.
	Saved    bool
	Schedule string
	// On is true when its timer or crontab entry is in place, and How is
	// "timer" or "cron".
	On  bool
	How string
	// Shadow writes what it would alert on and sends nothing. Notify also
	// shows each alert as a desktop notification.
	Shadow bool
	Notify bool
}

// AlertsPath is where the scheduled check writes the alerts it sends, one
// line each with the time in front. Shadow checks write to their own file.
func AlertsPath(o Options) string {
	return filepath.Join(o.book().Dir, "health", "alerts.log")
}

// ReadCheck is the background check's settings. It only reads.
func ReadCheck(o Options) CheckSettings {
	b := o.book()
	r, err := b.Load(CheckRecipe)
	if err != nil {
		return CheckSettings{}
	}
	cs := CheckSettings{Saved: true, Schedule: r.Schedule}
	if len(r.Steps) > 0 && len(r.Steps[0].Run) > 0 {
		for _, a := range r.Steps[0].Run[0].Argv {
			switch a {
			case "--shadow":
				cs.Shadow = true
			case "--notify":
				cs.Notify = true
			}
		}
	}
	if b.Exec != nil {
		cs.On, cs.How = loom.Scheduled(context.Background(), b.Exec, CheckRecipe)
	}
	return cs
}

// checkArgv is the scheduled check's command line.
func checkArgv(prog, alerts string, shadow, notify bool) []string {
	argv := []string{prog, "check", "--log", alerts}
	if shadow {
		argv = append(argv, "--shadow")
	}
	if notify {
		argv = append(argv, "--notify")
	}
	return argv
}

// SetCheckModes changes whether the scheduled check only shadows, and
// whether it notifies. The schedule runs the saved recipe, so the change
// takes effect on its next run, with no change to the timer or crontab.
func SetCheckModes(o Options, shadow, notify bool) error {
	b := o.book()
	r, err := b.Load(CheckRecipe)
	if err != nil {
		return errors.New("the background check has not been set up yet")
	}
	if len(r.Steps) == 0 || len(r.Steps[0].Run) == 0 || len(r.Steps[0].Run[0].Argv) == 0 {
		return errors.New("the background check's recipe has no command")
	}
	prog := r.Steps[0].Run[0].Argv[0]
	r.Steps[0].Run = action.Pipeline{{Argv: checkArgv(prog, AlertsPath(o), shadow, notify)}}
	return b.Save(r)
}

// UnscheduleCheck is the action that turns the background check off. Its
// settings are kept, so turning it on again keeps them.
func UnscheduleCheck(o Options) (action.Action, error) {
	b := o.book()
	r, err := b.Load(CheckRecipe)
	if err != nil {
		return action.Action{}, err
	}
	if loom.IsCron(r.Schedule) {
		return loom.UncronAction(context.Background(), b.Exec, CheckRecipe)
	}
	return loom.UnscheduleAction(CheckRecipe), nil
}

// Alert is one line the check wrote.
type Alert struct {
	When time.Time
	Text string
	// Shadow is true for a line a shadow check wrote: it was not sent.
	Shadow bool
}

// RecentAlerts are the last n alerts sent and the last n a shadow check
// would have sent, newest first. Lines without a time are left out.
func RecentAlerts(o Options, n int) []Alert {
	var out []Alert
	for _, src := range []struct {
		path   string
		shadow bool
	}{{AlertsPath(o), false}, {shadowPath(o), true}} {
		out = append(out, readAlerts(src.path, src.shadow, n)...)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].When.After(out[j].When) })
	return out
}

func readAlerts(path string, shadow bool, n int) []Alert {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var all []Alert
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		stamp, text, ok := strings.Cut(sc.Text(), " ")
		if !ok {
			continue
		}
		t, err := time.Parse(time.RFC3339, stamp)
		if err != nil {
			continue
		}
		all = append(all, Alert{When: t, Text: text, Shadow: shadow})
	}
	if len(all) > n {
		all = all[len(all)-n:]
	}
	return all
}
