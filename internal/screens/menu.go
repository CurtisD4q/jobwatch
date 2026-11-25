package screens

import (
	"fmt"
	"strings"

	jobhealth "github.com/CurtisD4q/pmin-jobhealth"
	mint "github.com/CurtisD4q/pmin-kit"
	"github.com/CurtisD4q/pmin-kit/layouts"
)

// menuItem is one line of the menu: a screen, the key that opens it from
// anywhere, and what it is for in a few words.
type menuItem struct {
	key, name, what string
	open            func(Options) *mint.Screen
}

// menuGroups are the menu's sections, in the order a person meets them:
// watching the jobs, setting them up, then help and how it looks.
func menuGroups() []struct {
	name  string
	items []menuItem
} {
	return []struct {
		name  string
		items []menuItem
	}{
		{"Watch", []menuItem{
			{"o", "Overview", "your jobs, today and the check", Overview},
			{"d", "The day", "each job's run and the gap after it", Day},
			{"a", "Alerts", "what was sent, or would have been", Alerts},
			{"", "Re-spacing", "start times that would overlap less", Respace},
		}},
		{"Set up", []menuItem{
			{"n", "Add a job", "record a line of your crontab", AddJobs},
			{"i", "Import history", "runs from the journal or a log", Import},
			{"c", "Background check", "alerts on a schedule, or in shadow", BackgroundCheck},
		}},
		{"Help", []menuItem{
			{"?", "How this works", "jobwatch in five steps", Guide},
			{"s", "Keys and settings", "keys, where things are kept", Settings},
			{"", "Theme", "the colours of every Mint app", func(Options) *mint.Screen { return mint.ThemeScreen() }},
		}},
	}
}

// Menu is the first screen: every screen jobwatch has, grouped, with the key
// that opens it from anywhere, and beside it how things stand at a glance.
// On a narrow terminal the two stack.
func Menu(o Options) *mint.Screen {
	s := &mint.Screen{Icon: "✦", Title: "Menu", TopTitle: "Go to", TopWidth: 1.2, Hints: keyHints, Boxed: true}
	glance := &mint.Pane{Title: "At a glance", Beside: true, Width: 1}
	s.Panes = []*mint.Pane{glance}
	s.Refresh = func(s *mint.Screen) {
		var secs []mint.Section
		for _, g := range menuGroups() {
			var rows []mint.Row
			for _, it := range g.items {
				it := it
				key := it.key
				if key == "" {
					key = "⏎"
				}
				rows = append(rows, mint.Action(mint.ActionOpts{ID: "menu-" + it.name, Label: it.name,
					Get:        func() string { return key },
					OnActivate: func(ctx *mint.Ctx) { ctx.Push(it.open(o)) }}))
			}
			secs = append(secs, mint.Section{Name: g.name, Rows: rows})
		}
		s.Sections = secs
		_, rep, _, err := o.check()
		if err != nil {
			s.Banner = ""
			glance.Sections = []mint.Section{{Rows: wrapped("Could not read the runs: "+err.Error(), "")}}
			return
		}
		s.Banner = menuBanner(rep, o.checkSettings().On)
		glance.Sections = menuGlance(o, rep)
	}
	glance.Refresh = func(*mint.Pane) {}
	s.Status = func() []mint.Badge { return o.badges() }
	s.Poll = o.live()
	return o.withKeys(s)
}

// menuBanner is the overview's one thing most worth doing, or, before there
// are any jobs, where to start.
func menuBanner(rep jobhealth.Report, checkOn bool) string {
	if len(rep.Statuses) == 0 {
		return fitBanner("New here? Press ? for how this works, or n to add your first job.", "New here? ? for help, n to add a job.")
	}
	return overviewBanner(rep, checkOn, len(rep.Statuses))
}

// menuGlance is how things stand, in a few lines: the jobs, the check and
// the latest alert. Each opens the screen with the rest.
func menuGlance(o Options, rep jobhealth.Report) []mint.Section {
	problems, warnings := 0, 0
	for _, st := range rep.Statuses {
		switch st.Level {
		case jobhealth.Problem:
			problems++
		case jobhealth.Warn:
			warnings++
		}
	}
	jobs := "none yet"
	switch n := len(rep.Statuses); {
	case n == 0:
	case problems > 0:
		jobs = fmt.Sprintf("%d, %s", n, plural(problems, "problem"))
	case warnings > 0:
		jobs = fmt.Sprintf("%d, %s", n, plural(warnings, "warning"))
	default:
		jobs = fmt.Sprintf("%d, all fine", n)
	}
	cs := o.checkSettings()
	check := "off"
	if cs.On {
		check = "on"
		if cs.Shadow {
			check = "on, in shadow"
		}
	}
	last := o.checkLastText()
	alert := "none yet"
	if al := o.recentAlerts(1); len(al) > 0 {
		job, _, _ := strings.Cut(al[0].Text, ":")
		alert = al[0].When.Local().Format("15:04") + " " + job
		if al[0].Shadow {
			alert += " ◌"
		}
	}
	open := func(f func(Options) *mint.Screen) func(*mint.Ctx) {
		return func(ctx *mint.Ctx) { ctx.Push(f(o)) }
	}
	rows := []mint.Row{
		mint.Action(mint.ActionOpts{ID: "gl-jobs", Label: "Jobs", Alert: problems > 0, Get: func() string { return jobs }, OnActivate: open(Overview)}),
		mint.Action(mint.ActionOpts{ID: "gl-check", Label: "Check", Get: func() string { return check }, OnActivate: open(BackgroundCheck)}),
		mint.Action(mint.ActionOpts{ID: "gl-last", Label: "Last ran", Get: func() string { return last }, OnActivate: open(BackgroundCheck)}),
		mint.Action(mint.ActionOpts{ID: "gl-alert", Label: "Last alert", Get: func() string { return alert }, OnActivate: open(Alerts)}),
	}
	// The pane is narrower than the terminal, and on a narrow one it is
	// under the menu, so this is wrapped to a width that fits either way.
	about := []mint.Row{mint.Spacer()}
	for _, l := range layouts.Wrap("jobwatch reads how your jobs have been running and says how each is doing, before a slow one runs into the next. It never changes a schedule.", 32) {
		about = append(about, mint.Note("  "+l))
	}
	about = append(about, mint.Spacer(), mint.Label(mint.StyleCaption, "  jobwatch "+o.version()))
	return []mint.Section{{Rows: rows}, {Rows: about}}
}
