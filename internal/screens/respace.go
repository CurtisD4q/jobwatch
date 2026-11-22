package screens

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/CurtisD4q/jobwatch/internal/jw"
	mint "github.com/CurtisD4q/mint-kit"
	loom "github.com/CurtisD4q/mint-loom"
	"github.com/CurtisD4q/mint-loom/health"
)

var respaceWindows = []time.Duration{30 * time.Minute, time.Hour, 2 * time.Hour, 4 * time.Hour}

// Respace suggests start times that make jobs overlap less, with the
// effect the health check expects. Nothing is applied: for a job recorded
// from a crontab line, the line with its new time can be copied, to paste
// in by hand.
func Respace(o Options) *mint.Screen {
	window := 2 * time.Hour
	scr := &mint.Screen{Icon: "⇄", Title: "Re-spacing"}
	scr.Refresh = func(s *mint.Screen) {
		jobs, err := health.FromBook(context.Background(), o.Book, o.now())
		if err != nil {
			s.Sections = []mint.Section{{Rows: []mint.Row{mint.Note("Could not read the runs: " + err.Error())}}}
			return
		}
		plan := jw.Respace(jobs, window, o.now())
		top := []mint.Row{mint.Action(mint.ActionOpts{ID: "window", Label: "Move a start by up to",
			Get: func() string { return shortDur(window) },
			OnActivate: func(ctx *mint.Ctx) {
				for i, w := range respaceWindows {
					if w == window {
						window = respaceWindows[(i+1)%len(respaceWindows)]
						return
					}
				}
				window = respaceWindows[0]
			}})}
		var lines map[string]loom.CronLine
		if o.Book.Exec != nil {
			if cls, err := loom.CronLines(context.Background(), o.Book.Exec); err == nil {
				lines = map[string]loom.CronLine{}
				for _, cl := range cls {
					if cl.Recorded != "" {
						lines[cl.Recorded] = cl
					}
				}
			}
		}
		var moves []mint.Row
		for i, m := range plan.Moves {
			m := m
			from, to := localClock(m.From, o.now()), localClock(m.To, o.now())
			cl, have := lines[m.Job]
			newLine, ok := "", false
			if have {
				newLine, ok = shiftedLine(cl, minutesBetween(m.From, m.To))
			}
			hint := "move by hand"
			if ok {
				hint = "copy the new line"
			}
			moves = append(moves, mint.Action(mint.ActionOpts{ID: fmt.Sprintf("m%d", i), Label: m.Job + "  " + from + " → " + to,
				Get: func() string { return hint },
				OnActivate: func(ctx *mint.Ctx) {
					if !ok {
						ctx.Message(m.Job, "Move its start from "+from+" to "+to+" where it is scheduled. Jobwatch does not change schedules.")
						return
					}
					ctx.Copy(newLine, "the new crontab line")
				}}))
		}
		if len(moves) == 0 {
			moves = []mint.Row{mint.Note("No move reduces the overlap.")}
		}
		about := append(wrapped("Overlap now: "+plan.Before+". After the moves: "+plan.After+".", ""), wrapped(plan.Effect, "")...)
		about = append(about, wrapped("Nothing is changed. Copy a line and replace the old one with crontab -e, or move the timer by hand.", "")...)
		s.Sections = []mint.Section{{Rows: top}, {Name: "Suggested moves", Rows: moves}, {Name: "Effect", Rows: about}}
	}
	return scr
}

func shortDur(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%d min", int(d/time.Minute))
	}
	return fmt.Sprintf("%d h", int(d/time.Hour))
}

// hhmm reads "HH:MM" as minutes after midnight.
func hhmm(s string) (int, bool) {
	h, m, ok := strings.Cut(s, ":")
	if !ok {
		return 0, false
	}
	hi, e1 := strconv.Atoi(h)
	mi, e2 := strconv.Atoi(m)
	if e1 != nil || e2 != nil {
		return 0, false
	}
	return hi*60 + mi, true
}

// minutesBetween is how far to is after from, the short way round the clock.
func minutesBetween(from, to string) int {
	a, ok1 := hhmm(from)
	b, ok2 := hhmm(to)
	if !ok1 || !ok2 {
		return 0
	}
	d := (b - a + 1440) % 1440
	if d > 720 {
		d -= 1440
	}
	return d
}

// localClock is a UTC "HH:MM" as the local time of day.
func localClock(utc string, now time.Time) string {
	m, ok := hhmm(utc)
	if !ok {
		return utc
	}
	y, mo, d := now.UTC().Date()
	return time.Date(y, mo, d, m/60, m%60, 0, 0, time.UTC).Local().Format("15:04")
}

// shiftedLine is a crontab line with its start moved by shift minutes. It
// is only made for a plain minute and hour, as in 30 2 * * *, and only when
// the move stays on the same day, so the day fields keep meaning the same.
func shiftedLine(cl loom.CronLine, shift int) (string, bool) {
	f := strings.Fields(cl.Schedule)
	if len(f) != 5 {
		return "", false
	}
	mi, e1 := strconv.Atoi(f[0])
	hour, e2 := strconv.Atoi(f[1])
	if e1 != nil || e2 != nil || mi < 0 || mi > 59 || hour < 0 || hour > 23 {
		return "", false
	}
	t := hour*60 + mi + shift
	if t < 0 || t >= 1440 {
		return "", false
	}
	f[0], f[1] = strconv.Itoa(t%60), strconv.Itoa(t/60)
	return strings.Join(f, " ") + " " + cl.Command, true
}
