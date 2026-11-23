package jw

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	jobhealth "github.com/CurtisD4q/pmin-jobhealth"
)

func runsAt(hour, min int, dur time.Duration, n int) []jobhealth.Run {
	var out []jobhealth.Run
	first := time.Date(2026, 1, 1, hour, min, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		s := first.AddDate(0, 0, i)
		out = append(out, jobhealth.Run{Start: s, End: s.Add(dur), Elapsed: dur.Seconds(), OK: true, By: "schedule"})
	}
	return out
}

func TestRespacePrintsAPlanAndChangesNothing(t *testing.T) {
	jobs := []jobhealth.Job{
		{Name: "backup", Schedule: "0 2 * * *", Period: 24 * time.Hour, Runs: runsAt(2, 0, 90*time.Minute, 5)},
		{Name: "prune", Schedule: "30 2 * * *", Period: 24 * time.Hour, Runs: runsAt(2, 30, 20*time.Minute, 5)},
	}
	now := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	plan := Respace(jobs, 2*time.Hour, now)
	if plan.Applied {
		t.Fatal("applied a change")
	}
	if plan.Before == plan.After || plan.After != "0 min" {
		t.Fatalf("%+v", plan)
	}
	if len(plan.Moves) == 0 || !strings.Contains(plan.Effect, "Health check") {
		t.Fatalf("%+v", plan)
	}

	dir := t.TempDir()
	before, _ := os.ReadDir(dir)
	o := Options{Dir: dir, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	if code := Main([]string{"respace", "--window", "2h"}, o); code != 0 {
		t.Fatal(code)
	}
	out := o.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(out, "Nothing was changed.") {
		t.Fatalf("%s", out)
	}
	after, _ := os.ReadDir(dir)
	if len(before) != len(after) {
		t.Fatal("respace wrote a file")
	}
}
