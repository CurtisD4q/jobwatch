package screens

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/CurtisD4q/jobwatch/internal/jw"
	mint "github.com/CurtisD4q/mint-kit"
	"github.com/CurtisD4q/mint-tools/action"
)

// timerService is a service a systemd timer starts, in the system's
// journal or the person's own.
type timerService struct {
	Unit string
	User bool
}

// timerServices lists the services timers start: what is worth importing.
// It only reads; where systemctl cannot answer (a container, no systemd),
// the list is empty and a service can still be named by hand.
func timerServices(exec action.Executor) []timerService {
	var out []timerService
	seen := map[string]bool{}
	for _, user := range []bool{false, true} {
		argv := []string{"systemctl", "list-timers", "--all", "--no-legend"}
		if user {
			argv = []string{"systemctl", "--user", "list-timers", "--all", "--no-legend"}
		}
		r := exec.Run(context.Background(), action.Pipeline{{Argv: argv, Timeout: 10 * time.Second}})
		if r.Err != nil || r.Exit != 0 {
			continue
		}
		for _, line := range strings.Split(string(r.Stdout), "\n") {
			f := strings.Fields(line)
			if len(f) == 0 {
				continue
			}
			unit := f[len(f)-1]
			key := fmt.Sprint(user, unit)
			if strings.HasSuffix(unit, ".service") && !seen[key] {
				seen[key] = true
				out = append(out, timerService{Unit: unit, User: user})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Unit < out[j].Unit })
	return out
}

// Import reads history jobwatch did not record: a timer's service from the
// journal, or an rsync, rclone or cron log. The runs are kept apart from
// recorded ones and marked lower confidence; the log lines are not kept.
func Import(o Options) *mint.Screen {
	scr := &mint.Screen{Icon: "⇣", Title: "Import history"}
	var services []timerService
	listed := false
	scr.Refresh = func(s *mint.Screen) {
		if !listed {
			services, listed = timerServices(o.Book.Exec), true
		}
		var journal []mint.Row
		for _, ts := range services {
			ts := ts
			where := "system"
			if ts.User {
				where = "yours"
			}
			journal = append(journal, mint.Action(mint.ActionOpts{ID: "imp-" + where + ts.Unit, Label: strings.TrimSuffix(ts.Unit, ".service"),
				Get:        func() string { return where },
				OnActivate: func(ctx *mint.Ctx) { importJournal(ctx, o, ts.Unit, ts.User) }}))
		}
		if len(journal) == 0 {
			journal = []mint.Row{mint.Note("No timers found here. Name a service below.")}
		}
		journal = append(journal, mint.Action(mint.ActionOpts{ID: "imp-named", Label: "A service by name…",
			OnActivate: func(ctx *mint.Ctx) {
				unit, ok := ctx.Prompt("Import a service", "Its name, such as backup.service", "")
				if !ok || strings.TrimSpace(unit) == "" {
					return
				}
				where, ok := ctx.Choose("Whose journal?", []string{"The system's", "Yours (a user service)"}, "")
				if !ok {
					return
				}
				importJournal(ctx, o, unit, strings.HasPrefix(where, "Yours"))
			}}))
		var logs []mint.Row
		for _, kind := range []string{"rsync", "rclone", "cron"} {
			kind := kind
			logs = append(logs, mint.Action(mint.ActionOpts{ID: "imp-log-" + kind, Label: kind + " log…",
				OnActivate: func(ctx *mint.Ctx) { importLog(ctx, o, kind) }}))
		}
		about := wrapped("Imported runs are kept apart from the runs jobwatch records, and marked lower confidence (◌). A cron log has start times only, so it shows missed runs and nothing about how long they took. What the logs say is not kept.", "")
		s.Sections = []mint.Section{
			{Name: "From the journal, last 90 days", Rows: journal},
			{Name: "From a log file", Rows: logs},
			{Rows: about},
		}
	}
	return scr
}

func importJournal(ctx *mint.Ctx, o Options, unit string, user bool) {
	n, err := jw.ImportJournalFrom(o.jwOpts(), unit, "90d", user)
	o.changed()
	if err != nil {
		ctx.ToastError(err.Error())
		return
	}
	ctx.Toast(fmt.Sprintf("Imported %s of %s.", plural(n, "run"), strings.TrimSuffix(unit, ".service")))
}

func importLog(ctx *mint.Ctx, o Options, kind string) {
	path, ok := ctx.Prompt("Import "+kind+" history", "The log file's path", "")
	path = strings.TrimSpace(path)
	if !ok || path == "" {
		return
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[2:])
		}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		ctx.ToastError(err.Error())
		return
	}
	def := suggestName(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	name, ok := ctx.Prompt("Name the job", "A short name: lower-case letters, numbers and dashes", def)
	name = strings.TrimSpace(name)
	if !ok || name == "" {
		return
	}
	n, err := jw.ImportLog(o.jwOpts(), kind, name, string(b), o.now())
	o.changed()
	if err != nil {
		ctx.ToastError(err.Error())
		return
	}
	ctx.Toast(fmt.Sprintf("Imported %s of %s.", plural(n, "run"), name))
}
