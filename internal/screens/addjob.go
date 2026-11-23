package screens

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"

	mint "github.com/CurtisD4q/pmin-kit"
	loom "github.com/CurtisD4q/pmin-loom"
)

// AddJobs lists the lines of the person's own crontab and records the runs
// of the one they choose, where it is. Each change is shown in full and
// confirmed before it is written, and can be undone. Lines that need the
// shell are explained, not changed.
func AddJobs(o Options) *mint.Screen {
	scr := &mint.Screen{Icon: "+", Title: "Add a job"}
	scr.Refresh = func(s *mint.Screen) {
		lines, err := o.cronLines()
		if err != nil {
			s.Sections = []mint.Section{{Rows: []mint.Row{mint.Note(err.Error())}}}
			return
		}
		intro := wrapped("Choose a line of your crontab to record its runs. It keeps its schedule, and its output and exit code still reach cron.", "")
		if why := o.cannotChange(); why != "" {
			intro = append(intro, wrapped(capital(why)+".", "")...)
		}
		var rows []mint.Row
		for i, cl := range lines {
			cl := cl
			rows = append(rows, mint.Action(mint.ActionOpts{
				ID:    "cron-" + string(rune('a'+i%26)) + cl.Schedule,
				Label: cl.Schedule + "  " + shorten(commandOf(cl), 48),
				Get:   func() string { return lineState(cl) },
				OnActivate: func(ctx *mint.Ctx) {
					switch {
					case cl.Recorded != "":
						stopRecording(ctx, o, cl)
					case cl.Shell != "":
						ctx.Message("Not recorded", "This line cannot be recorded as it is: "+cl.Shell+".\n\n"+
							"Cron runs the line with /bin/sh, and a recorded command runs without a shell. Put the command in a script, schedule the script, and record that line.")
					case o.cannotChange() != "":
						ctx.Message("Cannot record from here", capital(o.cannotChange())+".")
					default:
						startRecording(ctx, o, cl)
					}
				},
			}))
		}
		if len(rows) == 0 {
			rows = []mint.Row{mint.Note("Your crontab has no job lines.")}
		}
		s.Sections = []mint.Section{{Rows: intro}, {Name: "Your crontab", Rows: rows}}
	}
	return o.withKeys(scr)
}

func lineState(cl loom.CronLine) string {
	switch {
	case cl.Recorded != "":
		return "recorded as " + cl.Recorded
	case cl.Shell != "":
		return "needs the shell"
	}
	return "not recorded"
}

// commandOf is the command a line runs, without the recording in front.
func commandOf(cl loom.CronLine) string {
	if cl.Recorded == "" {
		return cl.Command
	}
	if _, after, ok := strings.Cut(cl.Command, " -- "); ok {
		return after
	}
	return cl.Command
}

func shorten(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

var nameJunk = regexp.MustCompile(`[^a-z0-9]+`)

// suggestName is a job name from a command: the program and, when it is a
// plain word, what it is asked to do ("restic-backup", "updatedb").
func suggestName(cmd string) string {
	w := strings.Fields(cmd)
	if len(w) == 0 {
		return "job"
	}
	base := strings.TrimSuffix(filepath.Base(w[0]), filepath.Ext(w[0]))
	parts := []string{base}
	if len(w) > 1 && regexp.MustCompile(`^[a-z][a-z-]*$`).MatchString(w[1]) {
		parts = append(parts, w[1])
	}
	name := strings.Trim(nameJunk.ReplaceAllString(strings.ToLower(strings.Join(parts, "-")), "-"), "-")
	if name == "" {
		name = "job"
	}
	if len(name) > 40 {
		name = strings.TrimRight(name[:40], "-")
	}
	return name
}

func startRecording(ctx *mint.Ctx, o Options, cl loom.CronLine) {
	name, ok := ctx.Prompt("Record this job", "A short name for it: lower-case letters, numbers and dashes", suggestName(cl.Command))
	name = strings.TrimSpace(name)
	if !ok || name == "" {
		return
	}
	if _, err := o.Book.Load(name); err == nil {
		ctx.ToastError("a recipe is already called " + name)
		return
	}
	a, err := loom.RecordCronAction(context.Background(), o.Book.Exec, cl.Raw, name, o.Program)
	if err != nil {
		ctx.ToastError(err.Error())
		return
	}
	o.do(ctx, a)
}

func stopRecording(ctx *mint.Ctx, o Options, cl loom.CronLine) {
	a, err := loom.UnrecordCronAction(context.Background(), o.Book.Exec, cl.Raw)
	if err != nil {
		ctx.ToastError(err.Error())
		return
	}
	o.do(ctx, a)
}
