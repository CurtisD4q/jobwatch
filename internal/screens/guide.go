package screens

import (
	mint "github.com/CurtisD4q/pmin-kit"
)

// Guide is the short orientation: what jobwatch does, in a few steps, in
// plain words.
func Guide(o Options) *mint.Screen {
	var rows []mint.Row
	add := func(head, body string) {
		rows = append(rows, mint.Label(mint.StyleHeading, head))
		rows = append(rows, wrapped(body, "")...)
		rows = append(rows, mint.Spacer())
	}
	add("What it is for", "Backups and syncs usually fail slowly: the data grows, runs take longer, the gap before the next job shrinks, until one runs into the next or is skipped. jobwatch reads how your jobs have been running and says, in a word, how each is doing, before that happens. It never changes a schedule.")
	add("Keys", "m the menu, o Overview, d the day, a alerts, c the background check, n add a job, i import history, s keys and settings, ? this page, r read everything again. They work on every screen.")
	add("1. Add your jobs", "Add a job lists your crontab: choose a line, and jobwatch shows the one change that records its runs, before and after, and changes it when you agree, with undo. The job still runs as before. Or put "+o.command()+" record NAME -- in front of a command yourself.")
	add("2. Bring in what already happened", "Import history reads a timer's service from the journal, or an rsync, rclone or cron log, so there is something to go on from the first day. Those runs are marked ◌, lower confidence.")
	add("3. Try the check in shadow", "Background check runs the check on a schedule. In shadow it writes what it would have sent and sends nothing: leave it for a week or two and compare with what really happened, then switch it to sending.")
	add("4. Read the overview", "Overview is the jobs, worst first, today's starts, and the check. A missed or failed run shows at once. Slowing down, running into the next job, and a step to a new level need some runs first, and say how sure they are.")
	add("5. Open a job for the evidence", "A job's screen has every finding with the numbers and the test behind it, its recent runs, and settings: mute a finding you know about, or delete the history.")
	add("What it does not do", "It suggests; it never moves or stops a job. Re-spacing shows start times that would overlap less, for you to change by hand. Contention between jobs is a hint, never proof: moving one job is the test.")
	rows = append(rows, wrapped("Keys: arrows move, Enter opens or changes, Tab moves between panes, b goes back, q quits. The mouse works too. Set MINT_KEYS=emacs or vim for those key styles.", "")...)
	return o.withKeys(&mint.Screen{Icon: "?", Title: "How this works", Hints: " ↑↓ scroll   b back   q quit ", Sections: []mint.Section{{Rows: rows}}})
}
