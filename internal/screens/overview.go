package screens

import (
	"fmt"
	"sort"
	"strings"
	"time"

	mint "github.com/CurtisD4q/mint-kit"
	"github.com/CurtisD4q/mint-loom/health"
)

// Overview is jobwatch's home: the jobs on the left, today in the middle
// and the background check on the right, side by side; on a narrow
// terminal they stack. The amber line says the one thing most worth doing,
// and the footer says how things stand.
func Overview(o Options) *mint.Screen {
	s := &mint.Screen{Icon: "✦", Title: "Overview", TopTitle: "Jobs", TopWidth: 1.1, Hints: keyHints, SectionSpacing: 1}
	today := &mint.Pane{Title: "Today", Beside: true, Width: 0.9}
	check := &mint.Pane{Title: "Background check", Beside: true, Width: 1.2}
	s.Panes = []*mint.Pane{today, check}
	s.Refresh = func(s *mint.Screen) {
		jobs, rep, _, err := o.check()
		if err != nil {
			s.Sections = []mint.Section{{Rows: []mint.Row{mint.Note("Could not read the runs: " + err.Error())}}}
			s.Banner = ""
			return
		}
		cs := o.checkSettings()
		s.Sections = overviewJobs(o, rep)
		s.Banner = overviewBanner(rep, cs.On, len(rep.Statuses))
		today.Sections = overviewToday(o, jobs)
		check.Sections = overviewCheck(o)
	}
	today.Refresh = func(*mint.Pane) {}
	check.Refresh = func(*mint.Pane) {}
	s.Status = func() []mint.Badge { return o.badges() }
	s.Poll = o.live()
	return o.withKeys(s)
}

// shortState is a job's state in a word or two, short enough never to be
// cut off beside its name.
func shortState(st health.Status) string {
	s := st.State
	if r := []rune(s); len(r) > 16 {
		s = string(r[:15]) + "…"
	}
	return s
}

func overviewJobs(o Options, rep health.Report) []mint.Section {
	if len(rep.Statuses) == 0 {
		return firstSteps(o)
	}
	statuses := append([]health.Status(nil), rep.Statuses...)
	// worst first, then by name, so what needs looking at is on top
	sort.SliceStable(statuses, func(i, j int) bool {
		if statuses[i].Level != statuses[j].Level {
			return statuses[i].Level > statuses[j].Level
		}
		return statuses[i].Job < statuses[j].Job
	})
	var rows []mint.Row
	for _, st := range statuses {
		st := st
		label := st.Job
		if st.Confidence == "low" {
			label += " ◌"
		}
		state := shortState(st)
		rows = append(rows, mint.Action(mint.ActionOpts{ID: "ov-" + st.Job, Label: label, Alert: st.Level == health.Problem,
			Get:        func() string { return state },
			OnActivate: func(ctx *mint.Ctx) { ctx.Push(Evidence(o, st.Job)) }}))
	}
	more := []mint.Row{
		mint.Action(mint.ActionOpts{ID: "ov-add", Label: "+ Add a job", OnActivate: func(ctx *mint.Ctx) { ctx.Push(AddJobs(o)) }}),
		mint.Action(mint.ActionOpts{ID: "ov-import", Label: "⇣ Import history", OnActivate: func(ctx *mint.Ctx) { ctx.Push(Import(o)) }}),
	}
	secs := []mint.Section{{Rows: rows}, {Rows: more}}
	for _, st := range statuses {
		if st.Confidence == "low" {
			secs = append(secs, mint.Section{Rows: []mint.Row{mint.Label(mint.StyleCaption, "◌ read from an outside log")}})
			break
		}
	}
	return secs
}

// firstSteps are the jobs pane when there are no jobs yet.
func firstSteps(o Options) []mint.Section {
	return []mint.Section{{Name: "No jobs yet", Rows: append(append(wrapped("jobwatch watches jobs it has recorded. Start here:", ""),
		mint.Action(mint.ActionOpts{ID: "fs-add", Label: "1. Add a job from your crontab", OnActivate: func(ctx *mint.Ctx) { ctx.Push(AddJobs(o)) }}),
		mint.Action(mint.ActionOpts{ID: "fs-import", Label: "2. Import history", OnActivate: func(ctx *mint.Ctx) { ctx.Push(Import(o)) }}),
		mint.Action(mint.ActionOpts{ID: "fs-check", Label: "3. Turn on the check, in shadow", OnActivate: func(ctx *mint.Ctx) { ctx.Push(BackgroundCheck(o)) }}),
		mint.Action(mint.ActionOpts{ID: "fs-guide", Label: "How this works", OnActivate: func(ctx *mint.Ctx) { ctx.Push(Guide(o)) }}),
	), wrapped("Or put "+o.command()+" record NAME -- in front of a command.", "")...)}}
}

// localStart is a bar's usual start (minutes after midnight, UTC) as a
// local time of day, on the day of now.
func localStart(min float64, now time.Time) string {
	y, m, d := now.UTC().Date()
	t := time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Add(time.Duration(min * float64(time.Minute))).Round(time.Minute)
	return t.Local().Format("15:04")
}

func overviewToday(o Options, jobs []health.Job) []mint.Section {
	now := o.now()
	var running []mint.Row
	for _, j := range jobs {
		if n := len(j.Runs); n > 0 && j.Runs[n-1].State == "running" {
			since := j.Runs[n-1].Start
			running = append(running, mint.Note(fmt.Sprintf("%s  %s, since %s", "●", j.Name, since.Local().Format("15:04"))))
		}
	}
	_, bars := health.DayTimeline(jobs, now)
	sort.SliceStable(bars, func(i, j int) bool { return bars[i].Start < bars[j].Start })
	var starts []mint.Row
	for _, b := range bars {
		if b.Period > 24*60+1 {
			continue // weekly and longer: not "today"
		}
		starts = append(starts, mint.Note(localStart(b.Start, now)+"  "+b.Name))
	}
	var secs []mint.Section
	if len(running) > 0 {
		secs = append(secs, mint.Section{Name: "Running", Rows: running})
	}
	if len(starts) == 0 {
		starts = []mint.Row{mint.Note("No daily job has enough runs yet.")}
	}
	secs = append(secs, mint.Section{Name: "Usual starts", Rows: starts})
	secs = append(secs, mint.Section{Rows: []mint.Row{
		mint.Action(mint.ActionOpts{ID: "ov-day", Label: "The day", Get: func() string { return "⏎" },
			OnActivate: func(ctx *mint.Ctx) { ctx.Push(Day(o)) }}),
		mint.Action(mint.ActionOpts{ID: "ov-respace", Label: "Re-spacing", Get: func() string { return "⏎" },
			OnActivate: func(ctx *mint.Ctx) { ctx.Push(Respace(o)) }}),
	}})
	return secs
}

func overviewCheck(o Options) []mint.Section {
	cs := o.checkSettings()
	state := "off"
	switch {
	case cs.On:
		state = "on, " + cs.Schedule
	case cs.Saved:
		state = "off"
	}
	mode := "sends alerts"
	if cs.Shadow {
		mode = "shadow"
	}
	last := "has not run"
	if r, ok := o.checkLastRun(); ok {
		last = ago(o.now().Sub(r.Start)) + " ago"
		if last == "now ago" {
			last = "just now"
		}
		if !r.OK && r.Skip == "" {
			last = "failed " + last
		}
	}
	rows := []mint.Row{
		mint.Action(mint.ActionOpts{ID: "ov-check", Label: "Check", Get: func() string { return state },
			OnActivate: func(ctx *mint.Ctx) { ctx.Push(BackgroundCheck(o)) }}),
		mint.Action(mint.ActionOpts{ID: "ov-last", Label: "Last ran", Get: func() string { return last },
			OnActivate: func(ctx *mint.Ctx) { ctx.Push(BackgroundCheck(o)) }}),
		mint.Action(mint.ActionOpts{ID: "ov-mode", Label: "Mode", Get: func() string { return mode },
			OnActivate: func(ctx *mint.Ctx) { ctx.Push(BackgroundCheck(o)) }}),
	}
	alerts := o.recentAlerts(5)
	// A pane is narrower than the terminal, so each alert is its time and
	// its job here; the whole alert is on Alerts, one key away.
	var recent []mint.Row
	for i, a := range alerts {
		job, _, _ := strings.Cut(a.Text, ":")
		mark := "sent"
		if a.Shadow {
			mark = "◌ shadow"
		}
		recent = append(recent, mint.Action(mint.ActionOpts{ID: fmt.Sprintf("ov-al-%d", i), Label: a.When.Local().Format("15:04") + "  " + job,
			Get:        func() string { return mark },
			OnActivate: func(ctx *mint.Ctx) { ctx.Push(Alerts(o)) }}))
	}
	if len(recent) == 0 {
		recent = []mint.Row{mint.Note("No alerts yet.")}
	}
	recent = append(recent, mint.Action(mint.ActionOpts{ID: "ov-alerts", Label: "All alerts", Get: func() string { return "⏎" },
		OnActivate: func(ctx *mint.Ctx) { ctx.Push(Alerts(o)) }}))
	return []mint.Section{{Rows: rows}, {Name: "Latest alerts", Rows: recent}}
}

// overviewBanner is the one thing most worth doing, or "" when all is well.
func overviewBanner(rep health.Report, checkOn bool, jobs int) string {
	var problems, names []string
	for _, st := range rep.Statuses {
		if st.Level == health.Problem {
			problems = append(problems, st.Job+" is "+st.State)
			names = append(names, st.Job)
		}
	}
	switch {
	case len(problems) == 1:
		return fitBanner("1 problem: "+problems[0], "1 problem: "+names[0])
	case len(problems) > 1:
		long := fmt.Sprintf("%d problems: %s", len(problems), strings.Join(problems, ", "))
		short := fmt.Sprintf("%d problems: %s", len(problems), strings.Join(names, ", "))
		return fitBanner(long, short, fmt.Sprintf("%d problems: %s and %d more", len(problems), names[0], len(names)-1))
	case jobs > 0 && !checkOn:
		return fitBanner("The background check is off: nothing will alert you. Press c to turn it on.", "The check is off: nothing will alert you (c).")
	}
	return ""
}

// badges are the footer: how the jobs stand, and the check.
func (o Options) badges() []mint.Badge {
	_, rep, _, err := o.check()
	if err != nil {
		return []mint.Badge{{Text: "cannot read runs", Kind: mint.BadgeError}}
	}
	problems, warnings := 0, 0
	for _, st := range rep.Statuses {
		switch st.Level {
		case health.Problem:
			problems++
		case health.Warn:
			warnings++
		}
	}
	var b []mint.Badge
	switch {
	case len(rep.Statuses) == 0:
		b = append(b, mint.Badge{Text: "no jobs yet", Kind: mint.BadgeInfo})
	case problems > 0:
		b = append(b, mint.Badge{Text: plural(problems, "problem"), Kind: mint.BadgeError})
	case warnings > 0:
		b = append(b, mint.Badge{Text: plural(warnings, "warning"), Kind: mint.BadgeWarn})
	default:
		b = append(b, mint.Badge{Text: "all fine", Kind: mint.BadgeOK})
	}
	cs := o.checkSettings()
	switch {
	case !cs.On:
		b = append(b, mint.Badge{Text: "check off", Kind: mint.BadgeWarn})
	case cs.Shadow:
		b = append(b, mint.Badge{Text: "shadow", Kind: mint.BadgeInfo})
	default:
		b = append(b, mint.Badge{Text: "check on", Kind: mint.BadgeOK})
	}
	return b
}

func plural(n int, w string) string {
	if n == 1 {
		return "1 " + w
	}
	return fmt.Sprintf("%d %ss", n, w)
}

// fitBanner is the first of the ways of saying it that fits on one line of
// this terminal, or the last.
func fitBanner(ways ...string) string {
	room := termCols() - 6
	for _, w := range ways {
		if len([]rune(w)) <= room {
			return w
		}
	}
	return ways[len(ways)-1]
}
