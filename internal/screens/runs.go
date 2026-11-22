package screens

import (
	"fmt"
	"math"
	"strings"
	"time"

	mint "github.com/CurtisD4q/mint-kit"
	loom "github.com/CurtisD4q/mint-loom"
)

const sparkBlocks = "▁▂▃▄▅▆▇█"

// sparkline draws values oldest to newest, one block each, as bars from
// zero to the longest (█), so lengths that are nearly the same look the
// same rather than being stretched to fill the height.
func sparkline(vals []float64) string {
	if len(vals) == 0 {
		return ""
	}
	hi := 0.0
	for _, v := range vals {
		hi = math.Max(hi, v)
	}
	blocks := []rune(sparkBlocks)
	var b strings.Builder
	for _, v := range vals {
		i := len(blocks) - 1
		if hi > 0 {
			i = int(math.Round(math.Max(v, 0) / hi * float64(len(blocks)-1)))
		}
		b.WriteRune(blocks[i])
	}
	return b.String()
}

// size says a number of bytes the way df -h does.
func size(b int64) string {
	if b <= 0 {
		return ""
	}
	f := float64(b)
	for _, u := range []string{"B", "kB", "MB", "GB", "TB"} {
		if f < 1000 || u == "TB" {
			if f < 10 && u != "B" {
				return fmt.Sprintf("%.1f %s", f, u)
			}
			return fmt.Sprintf("%.0f %s", f, u)
		}
		f /= 1000
	}
	return ""
}

func length(d time.Duration) string {
	switch {
	case d <= 0:
		return "—"
	case d < time.Minute:
		return d.Round(time.Second).String()
	case d < time.Hour:
		return fmt.Sprintf("%d min", int(d.Round(time.Minute)/time.Minute))
	}
	return fmt.Sprintf("%.1f h", d.Hours())
}

// result is how a run went, in a word or two.
func result(r loom.Run) string {
	switch {
	case r.Skip != "":
		return "skipped"
	case r.State == "running":
		return "running"
	case r.State == "lost":
		return "lost"
	case r.OK:
		return "done"
	}
	for _, s := range r.Steps {
		if s.Exit != 0 {
			return fmt.Sprintf("failed (exit %d)", s.Exit)
		}
	}
	return "failed"
}

// runRows are a job's recent runs, newest first, with a line drawing the
// lengths of the successful ones. Each run opens what it ran and the end of
// what it printed (kept unless the job stores no output).
func runRows(o Options, name string) []mint.Row {
	runs, err := o.Book.History(name, 30)
	if err != nil || len(runs) == 0 {
		return []mint.Row{mint.Note("No runs recorded.")}
	}
	var lens []float64
	var lo, hi time.Duration
	for i := len(runs) - 1; i >= 0; i-- { // oldest first
		r := runs[i]
		if !r.OK || r.Skip != "" {
			continue
		}
		d := r.Duration()
		lens = append(lens, d.Seconds())
		if lo == 0 || d < lo {
			lo = d
		}
		if d > hi {
			hi = d
		}
	}
	var rows []mint.Row
	if len(lens) >= 2 {
		rows = append(rows, mint.Note("Lengths, oldest to newest:"), mint.Note(fmt.Sprintf("%s  %s to %s", sparkline(lens), length(lo), length(hi))))
	}
	shown := runs
	if len(shown) > 12 {
		shown = shown[:12]
	}
	for i, r := range shown {
		r := r
		label := r.Start.Local().Format("Mon 2 Jan 15:04") + "  " + length(r.Duration())
		if s := size(r.Bytes); s != "" {
			label += "  " + s
		}
		res := result(r)
		rows = append(rows, mint.Action(mint.ActionOpts{ID: fmt.Sprintf("run-%d", i), Label: label, Alert: res != "done" && res != "running" && res != "skipped",
			Get:        func() string { return res },
			OnActivate: func(ctx *mint.Ctx) { ctx.Message(r.Start.Local().Format("Mon 2 Jan 15:04"), runDetail(r)) }}))
	}
	return rows
}

// runDetail is one run in full: how it went, each step's command and exit,
// and the end of what a failed step printed.
func runDetail(r loom.Run) string {
	var b strings.Builder
	b.WriteString(r.Summary() + "\n")
	if r.By != "" {
		b.WriteString("Started by: " + r.By + "\n")
	}
	if s := size(r.Bytes); s != "" {
		fmt.Fprintf(&b, "Data: %s", s)
		if r.Files > 0 {
			fmt.Fprintf(&b, " in %d files", r.Files)
		}
		b.WriteString("\n")
	}
	if r.Confidence == "low" {
		b.WriteString("Read from an outside log, so lower confidence.\n")
	}
	for _, s := range r.Steps {
		fmt.Fprintf(&b, "\n%s: %s\n", s.Label, s.Command)
		if s.Exit != 0 {
			fmt.Fprintf(&b, "Exit %d\n", s.Exit)
		}
		if s.Error != "" {
			b.WriteString(s.Error + "\n")
		}
		if out := strings.TrimSpace(s.Output); out != "" && (s.Exit != 0 || s.Error != "") {
			lines := strings.Split(out, "\n")
			if len(lines) > 12 {
				lines = lines[len(lines)-12:]
			}
			b.WriteString("It printed (the end):\n" + strings.Join(lines, "\n") + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}
