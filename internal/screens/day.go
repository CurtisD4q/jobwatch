package screens

import (
	"fmt"
	"strings"

	mint "github.com/CurtisD4q/mint-kit"
	"github.com/CurtisD4q/mint-loom/health"
)

// Day is the 24-hour screen: each job's usual run, the slow tail, and the
// gap before the next job, drawn at the width of the terminal.
//
// Each job is two rows, a name and the bar under it, and the name is the
// row that is chosen, so opening a job's evidence from the day is the same
// action as opening it from the list.
//
// The bar is the one health.Day draws, only wider: two columns an hour is
// 48 columns, which leaves the right half of a normal terminal empty. Here
// it is drawn at whatever width the screen has, so the gap before the next
// job is a distance you can see rather than a sentence to read. The usual
// run is solid, the slow tail lighter, and the gap before the next job is
// drawn out to where that job starts.
func Day(o Options) *mint.Screen {
	scr := &mint.Screen{Icon: "◷", Title: "The day"}
	scr.Poll = o.live()
	scr.Refresh = func(s *mint.Screen) {
		now := o.now()
		jobs, _, _, err := o.check()
		if err != nil {
			s.Sections = []mint.Section{{Rows: []mint.Row{mint.Note("Could not read the runs: " + err.Error())}}}
			return
		}
		_, bars := health.DayTimeline(jobs, now)
		// The name's own column, then a space, then the bar: the bar gets
		// the rest of the terminal, so a wider window shows a wider day.
		indent, width := dayLayout(termCols())
		pad := strings.Repeat(" ", indent)
		rows := []mint.Row{mint.Note(pad + dayAxis(width))}
		for _, b := range bars {
			b := b
			label := b.Name
			if b.LowConfidence {
				label += " · low confidence"
			}
			rows = append(rows, mint.Action(mint.ActionOpts{
				ID:    "jd-" + b.Name,
				Label: pivot(label, dayNameWidth),
				// The row is the name, and the bar is the line under it,
				// where it can be as wide as the terminal.
				OnActivate: func(ctx *mint.Ctx) {
					ctx.Push(Evidence(o, b.Name))
				},
			}))
			rows = append(rows, mint.Label(mint.StyleBody, pad+drawBar(b, width)))
			// What the bar cannot show: a gap too long to draw, or a job
			// that never recorded how long it runs, so there was no run
			// to draw at all.
			if said := longGap(b); said != "" {
				rows = append(rows, wrapped(said, pad)...)
			}
		}
		if len(bars) == 0 {
			rows = append(rows, mint.Note("No job has enough runs to draw."))
		}
		rows = append(rows, mint.Spacer())
		rows = append(rows, wrapped("█ usual run   ▓ slow runs   ─ gap before the next job   ┆ where the next job starts", "")...)
		s.Sections = []mint.Section{{Rows: rows}}
	}
	return o.withKeys(scr)
}

// dayLayout is where the day's bars start and how wide they are in a
// terminal this wide. A bar is drawn on the line under its job's name; on
// a wide screen it starts after the name's column, so the bars line up
// with the names, and on a narrow one it starts at the edge, so the day
// is never cut off. Eight columns go to the frame, its margins and the scroll bar.
func dayLayout(cols int) (indent, width int) {
	if cols <= 0 {
		cols = 80
	}
	content := cols - 8
	if w := content - dayNameWidth - 1; w >= dayCols {
		return dayNameWidth + 1, w
	}
	if content-1 < 24 {
		return 1, 24
	}
	return 1, content - 1
}

// BarWidth is how wide the day's bar is drawn in a terminal of this size:
// the screen less the name's column, and never narrower than the two
// columns an hour health.Day draws (so a small window shortens the names'
// column first, and the day last).
func BarWidth(_, cols int) int {
	if cols <= 0 {
		cols = 80
	}
	if w := cols - dayNameWidth - 1; w > dayCols {
		return w
	}
	return dayCols
}

// dayNameWidth is the column the jobs' names are written in.
const dayNameWidth = 25

// dayCols is the narrowest bar drawn: two columns an hour, the width
// health.Day uses. A screen narrower than that keeps this width rather
// than drawing a day too small to read.
const dayCols = health.DayWidth - 4

// drawBar is one job's bar across the day: midnight on the left, midnight
// on the right, as wide as the screen allows.
//
//	█  the usual run, from its start to its usual end
//	▓  the slow tail, out to its slowest recent run
//	─  the gap from there to where the next job starts
//	┆  where that next job starts
//
// A job whose slow tail reaches the next job's start has no gap to draw,
// and its tail runs straight into the mark: that is what a job running
// into another job looks like.
func drawBar(b health.Bar, width int) string {
	if width < dayCols {
		width = dayCols
	}
	bar := make([]rune, width)
	for i := range bar {
		bar[i] = ' '
	}
	// Painted in order of importance, so a mark that lands in the same
	// column as the run it interrupts is still seen: the gap first, then
	// the slow tail, then the usual run, then where the next job starts.
	if !b.Overrun {
		paint(bar, b.Slow, b.Slow+b.Gap, '─')
	}
	paint(bar, b.Usual, b.Slow, '▓')
	paint(bar, b.Start, b.Usual, '█')
	if b.NextName != "" {
		paint(bar, b.Next, b.Next+1, '┆')
	}
	return string(bar)
}

// paint marks the columns a span of the day touches, so a span that is
// shorter than a column still shows: a twenty-minute slow tail on a
// fourth-of-an-hour column is a cell, not nothing.
func paint(bar []rune, from, to float64, ch rune) {
	if len(bar) == 0 || to <= from {
		return
	}
	w := 1440.0 / float64(len(bar))
	// A start read back from the runs is 239.99999 minutes as often as it
	// is 240, and without the nudge it lands a column early, so a one-minute
	// mark would take two.
	const nudge = 1e-6
	first := int(from/w + nudge)
	last := int((to-0.001)/w + nudge)
	if last < first {
		last = first
	}
	for i := first; i <= last; i++ {
		for _, j := range []int{i, i + len(bar), i - len(bar)} {
			if j >= 0 && j < len(bar) {
				bar[j] = ch
			}
		}
	}
}

// dayAxis is the hours across the top, for a bar of the given width.
func dayAxis(width int) string {
	ax := make([]rune, width)
	for i := range ax {
		ax[i] = ' '
	}
	for _, h := range []int{0, 6, 12, 18} {
		put(ax, h*width/24, fmt.Sprintf("%02d", h))
	}
	put(ax, width-2, "24")
	return string(ax)
}

func put(ax []rune, at int, s string) {
	if at < 0 {
		at = 0
	}
	for i, r := range s {
		if at+i >= len(ax) {
			return
		}
		ax[at+i] = r
	}
}

// pivot puts a name in its column, keeping the end of it: the end of a
// name is what tells two jobs apart, and backup-db and backup-photos are
// the same for nineteen characters.
func pivot(s string, width int) string {
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	return "…" + string(r[len(r)-width+1:])
}

// longGap is what the bar cannot show about a job: a gap too long to draw
// as a distance (no job starts on this day), or that it never recorded how
// long it runs, so there was no run to draw.
func longGap(b health.Bar) string {
	switch {
	case b.Slow <= b.Start:
		if b.Period >= weekly {
			return "no end time recorded (weekly)"
		}
		return "no end time recorded"
	case b.NextName != "" && b.Gap < 12*60:
		// The bar shows this one: the gap runs out to the next job's mark.
		return ""
	case b.NextName != "" && b.Period >= weekly:
		return "no job starts within 12 hours (weekly)"
	case b.NextName != "":
		return "no job starts within 12 hours"
	}
	return ""
}

// sayDay is a job's line in words: what its slow runs leave before the
// next job starts, or that it runs past it. The drawn bar shows the gap
// itself, so this is for reading rather than looking.
func sayDay(b health.Bar, mark bool) string {
	switch {
	case b.Slow <= b.Start:
		return ""
	case b.Overrun:
		return "runs past " + b.NextName
	case mark:
		return ""
	case b.Gap >= 12*60:
		if b.Period >= weekly {
			return "no job starts within 12 hours (weekly)"
		}
		return "no job starts within 12 hours"
	case b.Gap > 0:
		return spanText(b.Gap) + " before " + b.NextName
	}
	return ""
}

// weekly is the cycle, in minutes, of a job that runs once a week. Its bar
// is drawn on the day it starts, but it is marked so nobody reads "no job
// starts within 12 hours" as a mistake.
const weekly = 7 * 24 * 60

// spanText is a gap in words: "5 min", "1.5 hours", "2 days".
func spanText(minutes float64) string {
	switch {
	case minutes < 60:
		return fmt.Sprintf("%.0f min", minutes)
	case minutes < 48*60:
		h := minutes / 60
		if h < 10 {
			return fmt.Sprintf("%.1f hours", h)
		}
		return fmt.Sprintf("%.0f hours", h)
	}
	return fmt.Sprintf("%.0f days", minutes/1440)
}
