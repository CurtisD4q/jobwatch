package jw

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	loom "github.com/CurtisD4q/mint-loom"
	"github.com/CurtisD4q/mint-loom/health"
)

// Move is one suggested change of a job's usual start. Nothing here is applied.
type Move struct {
	Job  string `json:"job"`
	From string `json:"from"`
	To   string `json:"to"`
}

// SpacePlan is a re-spacing suggestion. Applied is always false.
type SpacePlan struct {
	Moves   []Move `json:"moves"`
	Before  string `json:"before"`
	After   string `json:"after"`
	Applied bool   `json:"applied"`
	Effect  string `json:"effect"`
}

type spaceItem struct {
	job   health.Job
	start int
	dur   time.Duration
}

// Respace searches start times inside window of each job's usual start and
// returns the arrangement with the least overlap. It never changes a schedule.
// The effect line is what the health check, the same one the simulator
// replays, says before and after the move.
func Respace(jobs []health.Job, window time.Duration, now time.Time) SpacePlan {
	if window <= 0 {
		window = 2 * time.Hour
	}
	var items []spaceItem
	for _, j := range jobs {
		start, dur, ok := usual(j)
		if !ok {
			continue
		}
		items = append(items, spaceItem{j, start, dur})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].dur > items[j].dur })
	placed := map[string]int{}
	limit := int(window / time.Minute)
	step := 5
	if window < 5*time.Minute {
		step = 1
	}
	for _, it := range items {
		best := it.start
		bestScore := math.MaxFloat64
		for off := -limit; off <= limit; off += step {
			at := modMin(it.start + off)
			placed[it.job.Name] = at
			score := overlapScore(items, placed)
			closer := math.Abs(float64(off)) < math.Abs(float64(modMin(best-it.start+720)-720))
			if score < bestScore-0.5 || (math.Abs(score-bestScore) < 0.5 && (at == it.start || closer)) {
				best, bestScore = at, score
			}
		}
		placed[it.job.Name] = best
	}
	before := overlapScore(items, nil)
	after := overlapScore(items, placed)
	shifted := append([]health.Job(nil), jobs...)
	index := map[string]int{}
	for i, j := range shifted {
		index[j.Name] = i
	}
	var moves []Move
	for _, it := range items {
		to := placed[it.job.Name]
		if to == it.start {
			continue
		}
		moves = append(moves, Move{Job: it.job.Name, From: clockMin(it.start), To: clockMin(to)})
		if i, ok := index[it.job.Name]; ok {
			shifted[i].Runs = shiftRuns(it.job.Runs, to)
		}
	}
	if moves == nil {
		moves = []Move{}
	}
	return SpacePlan{
		Moves: moves, Before: spanMin(before), After: spanMin(after), Applied: false,
		Effect: effect(jobs, shifted, now),
	}
}

func effect(before, after []health.Job, now time.Time) string {
	b := health.Check(before, now)
	a := health.Check(after, now)
	return fmt.Sprintf("Health check before: %s. After the move: %s.", crowd(b), crowd(a))
}

func crowd(rep health.Report) string {
	for _, f := range rep.Findings {
		if f.Kind == health.Concurrency {
			return f.Says
		}
	}
	return "no three jobs run at once"
}

func usual(j health.Job) (start int, dur time.Duration, ok bool) {
	var mins, durs []float64
	for _, r := range j.Runs {
		if r.By == "hand" || r.Skip != "" || !r.End.After(r.Start) {
			continue
		}
		mins = append(mins, float64(r.Start.UTC().Hour()*60+r.Start.UTC().Minute()))
		durs = append(durs, r.End.Sub(r.Start).Seconds())
	}
	if len(durs) < 3 {
		return 0, 0, false
	}
	return int(math.Round(median(mins))) % 1440, time.Duration(median(durs) * float64(time.Second)), true
}

func overlapScore(items []spaceItem, placed map[string]int) float64 {
	type span struct{ a, b int }
	var spans []span
	for _, it := range items {
		at := it.start
		if placed != nil {
			if v, ok := placed[it.job.Name]; ok {
				at = v
			}
		}
		spans = append(spans, span{at, at + int(it.dur/time.Minute)})
	}
	var mins float64
	for i := 0; i < len(spans); i++ {
		for j := i + 1; j < len(spans); j++ {
			a := spans[i].a
			if spans[j].a > a {
				a = spans[j].a
			}
			b := spans[i].b
			if spans[j].b < b {
				b = spans[j].b
			}
			if b > a {
				mins += float64(b - a)
			}
		}
	}
	return mins
}

func shiftRuns(runs []loom.Run, minute int) []loom.Run {
	out := make([]loom.Run, len(runs))
	copy(out, runs)
	for i, r := range out {
		d := r.End.Sub(r.Start)
		s := r.Start.UTC()
		ns := time.Date(s.Year(), s.Month(), s.Day(), minute/60, minute%60, 0, 0, time.UTC)
		out[i].Start = ns
		if !r.End.IsZero() {
			out[i].End = ns.Add(d)
		}
	}
	return out
}

func modMin(m int) int {
	m %= 1440
	if m < 0 {
		m += 1440
	}
	return m
}

func clockMin(m int) string {
	m = modMin(m)
	return fmt.Sprintf("%02d:%02d", m/60, m%60)
}

func spanMin(m float64) string {
	if m < 1 {
		return "0 min"
	}
	return fmt.Sprintf("%.0f min", m)
}

func median(xs []float64) float64 {
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

func respaceCmd(args []string, o Options) int {
	windowText, rest := take(args, "--window")
	js, bad := flags(rest, map[string]bool{"--json": false})
	if bad || len(remain(rest)) > 0 {
		fmt.Fprintln(o.err(), "use: "+o.name()+" respace --window 2h [--json]")
		return 2
	}
	window := 2 * time.Hour
	if windowText != "" {
		d, err := time.ParseDuration(windowText)
		if err != nil || d <= 0 {
			fmt.Fprintln(o.err(), "the window is a duration such as 30m or 2h")
			return 2
		}
		window = d
	}
	jobs, err := health.FromBook(context.Background(), o.book(), o.now())
	if err != nil {
		fmt.Fprintln(o.err(), err)
		return 1
	}
	plan := Respace(jobs, window, o.now())
	if js["--json"] {
		return writeJSON(o, plan)
	}
	if len(plan.Moves) == 0 {
		fmt.Fprintln(o.out(), "No move reduces the overlap.")
	}
	for _, m := range plan.Moves {
		fmt.Fprintf(o.out(), "%s  %s -> %s\n", m.Job, m.From, m.To)
	}
	fmt.Fprintf(o.out(), "Overlap was %s, and would be %s.\n", plan.Before, plan.After)
	fmt.Fprintln(o.out(), plan.Effect)
	fmt.Fprintln(o.out(), "Nothing was changed.")
	return 0
}
