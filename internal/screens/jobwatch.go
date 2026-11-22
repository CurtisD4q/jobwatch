package screens

import (
	"fmt"
	mint "github.com/CurtisD4q/mint-kit"
	"github.com/CurtisD4q/mint-kit/layouts"
	"github.com/CurtisD4q/mint-loom/health"
	"time"
)

// Entries are jobwatch's screens, for its home list.
func Entries(o Options) []mint.HomeItem {
	if o.snap == nil {
		o.snap = newSnapshot()
	}
	return []mint.HomeItem{
		{Icon: "✦", Label: "Menu", Desc: "Every screen, with its key, and how things stand",
			Open: func() *mint.Screen { return Menu(o) }},
		{Icon: "✦", Label: "Overview", Desc: "The jobs, today, and the background check, side by side",
			Open: func() *mint.Screen { return Overview(o) }},
		{Icon: "✦", Label: "Jobs", Desc: "Each job in a word, with the last success and the worst finding",
			Open: func() *mint.Screen { return Jobs(o) }},
		{Icon: "◷", Label: "The day", Desc: "24 hours: each job's usual run and the gap before the next",
			Open: func() *mint.Screen { return Day(o) }},
		{Icon: "⇄", Label: "Re-spacing", Desc: "Start times that would make jobs overlap less; nothing is applied",
			Open: func() *mint.Screen { return Respace(o) }},
		{Icon: "+", Label: "Add a job", Desc: "Record the runs of a line of your crontab, where it is",
			Open: func() *mint.Screen { return AddJobs(o) }},
		{Icon: "⏲", Label: "Background check", Desc: "Turn the check that sends alerts on or off; shadow mode; notifications",
			Open: func() *mint.Screen { return BackgroundCheck(o) }},
		{Icon: "⇣", Label: "Import history", Desc: "Runs from the journal, or an rsync, rclone or cron log",
			Open: func() *mint.Screen { return Import(o) }},
		{Icon: "?", Label: "How this works", Desc: "What jobwatch does, in a few steps",
			Open: func() *mint.Screen { return Guide(o) }},
		{Icon: "⚙", Label: "Keys and settings", Desc: "The keys, where things are kept, and how it looks",
			Open: func() *mint.Screen { return Settings(o) }},
		{Icon: "!", Label: "Alerts", Desc: "What was sent, and what a shadow check would have sent",
			Open: func() *mint.Screen { return Alerts(o) }},
	}
}

// Jobs is jobwatch's first screen: one row a job, its state, when it last
// succeeded, and the worst finding, worst first. Selecting a job opens its
// evidence, and the day is one key away, so the two views share the same
// selection.
func Jobs(o Options) *mint.Screen {
	scr := &mint.Screen{Icon: "✦", Title: "Jobs"}
	scr.Poll = o.live()
	scr.Refresh = func(s *mint.Screen) {
		_, rep, _, err := o.check()
		if err != nil {
			s.Sections = []mint.Section{{Rows: []mint.Row{mint.Note("Could not read the runs: " + err.Error())}}}
			return
		}
		if len(rep.Statuses) == 0 {
			s.Sections = []mint.Section{{Rows: []mint.Row{
				mint.Note("No jobs yet."),
				mint.Note("Put this in front of a crontab line:"),
				mint.Note("  " + o.command() + " record NAME -- COMMAND"),
			}}}
			return
		}
		var rows []mint.Row
		for _, st := range rep.Statuses {
			st := st
			val := shortState(st)
			if !st.LastOK.IsZero() {
				val += " · " + ago(o.now().Sub(st.LastOK))
			}
			label := st.Job
			if st.Confidence == "low" {
				label += " ◌"
			}
			rows = append(rows, mint.Action(mint.ActionOpts{
				ID: "jw-" + st.Job, Label: label, Alert: st.Level == health.Problem,
				Get: func() string { return val },
				OnActivate: func(ctx *mint.Ctx) {
					ctx.Push(Evidence(o, st.Job))
				},
			}))
		}
		note := "Each job's state, and how long since it last succeeded."
		for _, st := range rep.Statuses {
			if st.Confidence == "low" {
				note += " ◌ read from an outside log, so lower confidence."
				break
			}
		}
		s.Sections = []mint.Section{{Name: "Jobs", Rows: rows}, {Rows: wrapped(note, "")}}
	}
	return o.withKeys(scr)
}

// Evidence is one job's screen: every finding about it, with the numbers
// behind it. It is the same for a job chosen from the list and from the day.
func Evidence(o Options, name string) *mint.Screen {
	scr := &mint.Screen{Icon: "✦", Title: name}
	scr.Poll = o.live()
	scr.Refresh = func(s *mint.Screen) {
		_, rep, mutes, err := o.check()
		if err != nil {
			s.Sections = []mint.Section{{Rows: []mint.Row{mint.Note("Could not read the runs: " + err.Error())}}}
			return
		}
		var status *health.Status
		for i := range rep.Statuses {
			if rep.Statuses[i].Job == name {
				status = &rep.Statuses[i]
			}
		}
		if status == nil {
			s.Sections = []mint.Section{{Rows: []mint.Row{mint.Note("No job called " + name + " is recorded.")}}}
			return
		}
		head := wrapped(describe(*status), "")
		if status.Confidence == "low" {
			head = append(head, wrapped("Read from an outside log, so this is lower confidence.", "")...)
		}
		var found []mint.Row
		for _, f := range rep.Findings {
			if f.Job != name {
				continue
			}
			tag := ""
			if ok, _ := health.Muted(mutes, f); ok {
				tag = " (muted)"
			}
			found = append(found, wrapped(f.Level.String()+tag+"  "+string(f.Kind)+": "+f.Says, "")...)
			for _, e := range f.Evidence {
				found = append(found, wrapped(e, "   ")...)
			}
		}
		if len(found) == 0 {
			found = []mint.Row{mint.Note("Nothing to report for this job.")}
		}
		s.Sections = []mint.Section{
			{Name: "Now", Rows: head},
			{Name: "Evidence", Rows: found},
			{Name: "Runs", Rows: runRows(o, name)},
			{Name: "Settings", Rows: settingsRows(o, name, rep.Findings, mutes)},
			{Name: "", Rows: []mint.Row{
				mint.Action(mint.ActionOpts{ID: "day", Label: "The day",
					Get:        func() string { return "⏎" },
					OnActivate: func(ctx *mint.Ctx) { ctx.Push(Day(o)) }}),
				mint.Note("It never changes a schedule itself."),
			}},
		}
	}
	return o.withKeys(scr)
}

// termCols is the terminal's width, which text wraps to. Tests set it to
// draw a screen at a chosen width.
var termCols = func() int {
	_, cols := mint.TermSize()
	return cols
}

// wrapped is text as notes wrapped to the terminal's width, each line after
// the first indented as the first is, so a finding stays readable on a
// narrow screen rather than being cut off.
func wrapped(text, indent string) []mint.Row {
	cols := termCols()
	if cols <= 0 {
		cols = 80
	}
	var rows []mint.Row
	for _, l := range layouts.Wrap(text, cols-8-len(indent)) {
		rows = append(rows, mint.Note(indent+l))
	}
	return rows
}

// ago is a time since, in a few characters: "40m", "5h", "3d".
func ago(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
