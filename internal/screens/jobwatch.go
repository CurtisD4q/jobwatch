package screens

import (
	"fmt"
	jobhealth "github.com/CurtisD4q/pmin-jobhealth"
	mint "github.com/CurtisD4q/pmin-kit"
	"github.com/CurtisD4q/pmin-kit/layouts"
	"sort"
	"strings"
	"time"
)

// Entries are jobwatch's screens, for its home list.
func Entries(o Options) []mint.HomeItem {
	if o.snap == nil {
		o.snap = newSnapshot()
	}
	// The launch menu: what jobwatch opens on. Each screen has its own
	// icon, its name, and a line on what it is for.
	return []mint.HomeItem{
		{Icon: "◧", Label: "Overview", Desc: "The jobs, today, and the background check, side by side",
			Open: func() *mint.Screen { return Overview(o) }},
		{Icon: "≡", Label: "Jobs", Desc: "Each job in a word, with the last success and the worst finding",
			Open: func() *mint.Screen { return Jobs(o) }},
		{Icon: "◷", Label: "Day", Desc: "24 hours: each job's usual run and the gap before the next",
			Open: func() *mint.Screen { return Day(o) }},
		{Icon: "!", Label: "Alerts", Desc: "What was sent, and what a shadow check would have sent",
			Open: func() *mint.Screen { return Alerts(o) }},
		{Icon: "⇄", Label: "Respace", Desc: "Start times that would make jobs overlap less; nothing is applied",
			Open: func() *mint.Screen { return Respace(o) }},
		{Icon: "+", Label: "Add job", Desc: "Record the runs of a line of your crontab, where it is",
			Open: func() *mint.Screen { return AddJobs(o) }},
		{Icon: "⇣", Label: "Import", Desc: "Runs from the journal, or an rsync, rclone or cron log",
			Open: func() *mint.Screen { return Import(o) }},
		{Icon: "◉", Label: "Check", Desc: "Turn the check that sends alerts on or off; shadow mode; notifications",
			Open: func() *mint.Screen { return BackgroundCheck(o) }},
		{Icon: "?", Label: "Guide", Desc: "What jobwatch does, in a few steps",
			Open: func() *mint.Screen { return Guide(o) }},
		{Icon: "⚙", Label: "Settings", Desc: "The keys, where things are kept, and how it looks",
			Open: func() *mint.Screen { return Settings(o) }},
	}
}

// HomeNote is the line under the launch menu's title: where to start, or
// the one thing most worth knowing. It names screens on the menu, not keys,
// as the menu has none of its own.
func HomeNote(o Options) string {
	_, rep, _, err := o.check()
	if err != nil {
		return "Could not read the runs: " + err.Error()
	}
	if len(rep.Statuses) == 0 {
		return "New here? Open Guide, or Add job to record your first one."
	}
	if note := overviewBanner(rep, true, len(rep.Statuses)); note != "" {
		return note
	}
	if !o.checkSettings().On {
		return "The background check is off: nothing will alert you. Open Check to turn it on."
	}
	return ""
}

// HomeStatus is the badges on the launch menu's footer: how the jobs are,
// and the background check.
func HomeStatus(o Options) func() []mint.Badge {
	return func() []mint.Badge { return o.badges() }
}

// HomeContext is the line after the trail at the top of every screen: the
// jobs, the check and the latest alert, in a few words.
func HomeContext(o Options) func() string {
	return func() string {
		_, rep, _, err := o.check()
		if err != nil {
			return "could not read the runs"
		}
		problems, warnings := 0, 0
		for _, st := range rep.Statuses {
			switch st.Level {
			case jobhealth.Problem:
				problems++
			case jobhealth.Warn:
				warnings++
			}
		}
		jobs := "no jobs yet"
		switch n := len(rep.Statuses); {
		case n == 0:
		case problems > 0:
			jobs = plural(n, "job") + ", " + plural(problems, "problem")
		case warnings > 0:
			jobs = plural(n, "job") + ", " + plural(warnings, "warning")
		default:
			jobs = plural(n, "job") + ", all fine"
		}
		cs := o.checkSettings()
		check := "check off"
		if cs.On {
			check = "check on"
			if cs.Shadow {
				check = "check in shadow"
			}
		}
		out := jobs + " · " + check
		if al := o.recentAlerts(1); len(al) > 0 {
			job, _, _ := strings.Cut(al[0].Text, ":")
			out += " · last alert " + al[0].When.Local().Format("15:04") + " " + job
		}
		return out
	}
}

// Jobs is jobwatch's first screen: one row a job, its state, when it last
// succeeded, and the worst finding, worst first. Selecting a job opens its
// evidence, and the day is one key away, so the two views share the same
// selection.
func Jobs(o Options) *mint.Screen {
	scr := &mint.Screen{Icon: "✦", Title: "Jobs", TopTitle: "Jobs", Boxed: true, TopWidth: 1.3}
	scr.Poll = o.live()
	var rep jobhealth.Report
	scr.TopInfo = func() string {
		if n := len(rep.Statuses); n > 0 {
			return plural(n, "job") + " · worst first"
		}
		return ""
	}
	details := &mint.Pane{Title: "Details", Beside: true, Width: 1}
	details.Refresh = func(p *mint.Pane) { p.Sections = jobCard(o, scr, rep) }
	scr.Panes = []*mint.Pane{details}
	scr.Refresh = func(s *mint.Screen) {
		var err error
		_, rep, _, err = o.check()
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
		statuses := worstFirst(rep.Statuses)
		cells := make([][]string, len(statuses))
		for i, st := range statuses {
			last := "never"
			if !st.LastOK.IsZero() {
				last = since(o.now().Sub(st.LastOK))
			}
			cells[i] = []string{jobLabel(st), shortState(st), last, fmt.Sprint(st.Runs)}
		}
		table := mint.Table(mint.TableOpts{
			ID:      "jobs",
			Headers: []string{"Job", "State", "Last success", "Runs"},
			Rows:    cells,
			Columns: []mint.Column{{}, {}, {Dim: true, Right: true}, {Dim: true, Right: true}},
			Mark:    func(i int, _ []string) mint.Mark { return jobMark(statuses[i]) },
			OnSelect: func(ctx *mint.Ctx, i int, _ []string) {
				ctx.Push(Evidence(o, statuses[i].Job))
			},
		})
		note := "Enter opens a job with the numbers behind it."
		for _, st := range rep.Statuses {
			if st.Confidence == "low" {
				note += " ◌ read from an outside log, so lower confidence."
				break
			}
		}
		s.Sections = []mint.Section{{Rows: []mint.Row{table}}, {Rows: wrapped(note, "")}}
	}
	return o.withKeys(scr)
}

// jobCard is the details of the job under the cursor: its state in a
// sentence, then what was found about it.
func jobCard(o Options, scr *mint.Screen, rep jobhealth.Report) []mint.Section {
	sel, ok := scr.Selected()
	if !ok || sel.Index < 0 {
		return []mint.Section{{Rows: []mint.Row{mint.Note("Move to a job to see it here.")}}}
	}
	statuses := worstFirst(rep.Statuses)
	if sel.Index >= len(statuses) {
		return nil
	}
	st := statuses[sel.Index]
	facts := []string{shortState(st) + " · " + plural(st.Runs, "run")}
	if !st.LastOK.IsZero() {
		facts = append(facts, "last succeeded "+since(o.now().Sub(st.LastOK)))
	}
	body := sentence(st.Says)
	if body == "" {
		body = sentence(st.State)
	}
	if st.Confidence == "low" {
		body += "\n\nRead from an outside log, so this is lower confidence."
	}
	var found []string
	for _, f := range rep.Findings {
		// The state already says the worst finding; list the others.
		if f.Job == st.Job && f.Says != st.Says {
			found = append(found, capital(f.Level.String())+": "+sentence(f.Says))
		}
	}
	if len(found) > 0 {
		body += "\n\n" + strings.Join(found, "\n\n")
	}
	body += "\n\nEnter on the job opens the runs and settings."
	return []mint.Section{{Rows: []mint.Row{mint.Card(mint.CardOpts{Key: st.Job, Title: st.Job, Facts: facts, Body: body})}}}
}

// worstFirst sorts jobs worst first, then by name, so what needs looking
// at is on top.
func worstFirst(in []jobhealth.Status) []jobhealth.Status {
	out := append([]jobhealth.Status(nil), in...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Level != out[j].Level {
			return out[i].Level > out[j].Level
		}
		return out[i].Job < out[j].Job
	})
	return out
}

// jobLabel is a job's name, marked ◌ when it is known only from an
// outside log.
func jobLabel(st jobhealth.Status) string {
	if st.Confidence == "low" {
		return st.Job + " ◌"
	}
	return st.Job
}

// jobMark is the mark beside a job in a list: what state it is in, at a
// glance.
func jobMark(st jobhealth.Status) mint.Mark {
	switch {
	case st.Level == jobhealth.Problem:
		return mint.Mark{Glyph: "!", Kind: mint.BadgeError}
	case st.Level == jobhealth.Warn:
		return mint.Mark{Glyph: "●", Kind: mint.BadgeWarn}
	case st.State == "new":
		return mint.Mark{Glyph: "○", Kind: mint.BadgeIdle}
	}
	return mint.Mark{Glyph: "✓", Kind: mint.BadgeOK}
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
		var status *jobhealth.Status
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
			if ok, _ := jobhealth.Muted(mutes, f); ok {
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
				mint.Action(mint.ActionOpts{ID: "day", Label: "Day",
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

// since is how long ago something was, for a list or a card: "just now",
// "40m ago", "3d ago".
func since(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	return ago(d) + " ago"
}

// sentence is s with a capital letter and a full stop.
func sentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if !strings.HasSuffix(s, ".") && !strings.HasSuffix(s, "!") && !strings.HasSuffix(s, "?") {
		s += "."
	}
	return capital(s)
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
