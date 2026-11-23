package jw

import (
	"fmt"

	jobhealth "github.com/CurtisD4q/pmin-jobhealth"
)

// timelineJSON is one day, midnight to midnight, in minutes of the day.
// A job appears when it has at least three recent successful runs by its
// schedule; a job recorded by hand, or one with fewer, is left out. The
// times are minutes of the day, 0 to 1440, and a run that ends after
// midnight has an end above 1440. Clock is the 24-hour axis, as text.
type timelineJSON struct {
	Clock string            `json:"clock"`
	Bars  []timelineBarJSON `json:"bars"`
}

type timelineBarJSON struct {
	Name string `json:"name"`
	// State is the job's word, so the same job reads the same here and
	// in the report: fine, overdue, failing, slowing, overrunning, new,
	// running, lost, changed.
	State string `json:"state"`
	// Period is the job's cycle in minutes, 1440 for a daily job and
	// 10080 for a weekly one. Two jobs with different cycles are not on
	// the same circle, so a weekly job is drawn as a bar, not compared
	// with a daily one.
	Period float64 `json:"period"`
	// Start is its usual start, Usual the end of its usual run, Slow the
	// end of its slowest recent run (95th percentile).
	Start float64 `json:"start"`
	Usual float64 `json:"usual"`
	Slow  float64 `json:"slow"`
	// Next is the start of the next job to start, and NextName its name.
	// Gap is the minutes from Slow to Next, and Overrun says Slow is at
	// or past Next. Next is -1 when no other job starts within 12 hours.
	Next     float64 `json:"next"`
	NextName string  `json:"nextName,omitempty"`
	Gap      float64 `json:"gap"`
	Overrun  bool    `json:"overrun"`
	// Says is the job's own line, from the report.
	Says string `json:"says,omitempty"`
	// LowConfidence is set when the job is known only from an outside log.
	LowConfidence bool `json:"lowConfidence,omitempty"`
}

func timeline(args []string, o Options) int {
	js, ok := jsonFlag(args)
	if !ok {
		fmt.Fprintln(o.err(), usage(o.name()))
		return 2
	}
	jobs, rep, err := load(o)
	if err != nil {
		fmt.Fprintln(o.err(), err)
		return 1
	}
	state := map[string]jobhealth.Status{}
	for _, s := range rep.Statuses {
		state[s.Job] = s
	}
	line, bars := jobhealth.DayTimeline(jobs, o.now())
	if js["--json"] {
		out := timelineJSON{Clock: line, Bars: []timelineBarJSON{}}
		for _, b := range bars {
			st := state[b.Name]
			out.Bars = append(out.Bars, timelineBarJSON{
				Name: b.Name, State: st.State, Period: b.Period,
				Start: b.Start, Usual: b.Usual, Slow: b.Slow,
				Next: b.Next, NextName: b.NextName, Gap: b.Gap, Overrun: b.Overrun,
				Says: st.Says, LowConfidence: b.LowConfidence,
			})
		}
		if writeJSON(o, out) != 0 {
			return 1
		}
		return exitOf(rep)
	}
	for _, l := range jobhealth.Day(jobs, o.now()) {
		fmt.Fprintln(o.out(), l)
	}
	return exitOf(rep)
}
