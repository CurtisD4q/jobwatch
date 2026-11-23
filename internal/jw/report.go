package jw

import (
	"context"
	"fmt"
	"time"

	jobhealth "github.com/CurtisD4q/pmin-jobhealth"
	"github.com/CurtisD4q/pmin-loom/health"
)

type jobJSON struct {
	Name       string    `json:"name"`
	State      string    `json:"state"`
	Level      string    `json:"level"`
	Says       string    `json:"says,omitempty"`
	LastOK     time.Time `json:"lastOK,omitempty"`
	Runs       int       `json:"runs"`
	Durations  []float64 `json:"durations,omitempty"`
	Confidence string    `json:"confidence,omitempty"`
}

type findingJSON struct {
	Job      string   `json:"job"`
	Other    string   `json:"other,omitempty"`
	Kind     string   `json:"kind"`
	Level    string   `json:"level"`
	Says     string   `json:"says"`
	Evidence []string `json:"evidence,omitempty"`
}

type reportJSON struct {
	Jobs     []jobJSON     `json:"jobs"`
	Findings []findingJSON `json:"findings"`
}

func load(o Options) ([]jobhealth.Job, jobhealth.Report, error) {
	b := o.book()
	jobs, err := health.FromBook(context.Background(), b, o.now())
	if err != nil {
		return nil, jobhealth.Report{}, err
	}
	return jobs, jobhealth.Check(jobs, o.now()), nil
}

func runLengths(j jobhealth.Job) []float64 {
	var out []float64
	for _, r := range j.Runs {
		if r.OK && r.Skip == "" && r.Duration() > 0 {
			out = append(out, r.Duration().Seconds())
		}
	}
	if len(out) > 30 {
		out = out[len(out)-30:]
	}
	return out
}

func asJSON(jobs []jobhealth.Job, rep jobhealth.Report, evidence bool) reportJSON {
	lengths := map[string][]float64{}
	for _, j := range jobs {
		if d := runLengths(j); len(d) > 0 {
			lengths[j.Name] = d
		}
	}
	out := reportJSON{}
	for _, s := range rep.Statuses {
		out.Jobs = append(out.Jobs, jobJSON{Name: s.Job, State: s.State, Level: s.Level.String(), Says: s.Says, LastOK: s.LastOK, Runs: s.Runs, Durations: lengths[s.Job], Confidence: s.Confidence})
	}
	for _, f := range rep.Findings {
		j := findingJSON{Job: f.Job, Other: f.Other, Kind: string(f.Kind), Level: f.Level.String(), Says: f.Says}
		if evidence {
			j.Evidence = f.Evidence
		}
		out.Findings = append(out.Findings, j)
	}
	if out.Jobs == nil {
		out.Jobs = []jobJSON{}
	}
	if out.Findings == nil {
		out.Findings = []findingJSON{}
	}
	return out
}

func report(args []string, o Options) int {
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
	if js["--json"] {
		if writeJSON(o, asJSON(jobs, rep, true)) != 0 {
			return 1
		}
		return exitOf(rep)
	}
	if len(rep.Statuses) == 0 {
		fmt.Fprintln(o.out(), "No jobs yet.")
		return 0
	}
	for _, s := range rep.Statuses {
		line := s.Job + "  " + s.State
		if s.Confidence == "low" {
			line += "  lower confidence"
		}
		if s.Says != "" {
			line += "  " + s.Says
		}
		fmt.Fprintln(o.out(), line)
	}
	for _, f := range rep.Findings {
		fmt.Fprintf(o.out(), "%s  %s  %s\n", f.Level, f.Kind, f.Says)
	}
	return exitOf(rep)
}

func forecast(args []string, o Options) int {
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
	var kept []jobhealth.Finding
	for _, f := range rep.Findings {
		if f.Kind == jobhealth.Overrun {
			kept = append(kept, f)
		}
	}
	rep.Findings = kept
	if js["--json"] {
		return writeJSON(o, asJSON(jobs, rep, true))
	}
	if len(kept) == 0 {
		fmt.Fprintln(o.out(), "No overrun is forecast.")
		return 0
	}
	for _, f := range kept {
		fmt.Fprintf(o.out(), "%s  %s\n", f.Job, f.Says)
	}
	return 0
}
