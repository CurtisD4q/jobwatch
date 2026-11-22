package jw

import (
	"context"
	"errors"
	"fmt"

	loom "github.com/CurtisD4q/mint-loom"
	"github.com/CurtisD4q/mint-tools/action"
)

func schedule(args []string, o Options) int {
	fs := newFlags("schedule")
	asJSON := fs.Bool("json", false, "")
	on := fs.Bool("on", false, "")
	undo := fs.Bool("undo", false, "")
	rest, err := parse(fs, args)
	if err != nil || (*on && *undo) {
		fmt.Fprintln(o.err(), usage(o.name()))
		return 2
	}
	js := map[string]bool{"--json": *asJSON, "--on": *on, "--undo": *undo}
	spec := ""
	if len(rest) == 1 {
		spec = rest[0]
	} else if len(rest) > 1 || (!js["--undo"] && len(rest) == 0) {
		fmt.Fprintln(o.err(), "use: "+o.name()+" schedule SPEC [--on | --undo]")
		return 2
	}
	if js["--undo"] {
		return scheduleUndo(o, js["--json"])
	}
	if js["--on"] {
		return scheduleOn(o, spec, js["--json"])
	}
	next, err := loom.NextRuns(context.Background(), o.book().Exec, spec, 3)
	if err != nil {
		fmt.Fprintln(o.err(), err)
		return 1
	}
	if js["--json"] {
		return writeJSON(o, map[string]any{"schedule": spec, "next": next})
	}
	if len(next) == 0 {
		fmt.Fprintln(o.out(), "No next run.")
		return 0
	}
	for _, n := range next {
		fmt.Fprintln(o.out(), n)
	}
	return 0
}

// ScheduleCheck builds the action that runs the check on spec, and its
// undo. It saves the recipe and does not run the action.
func ScheduleCheck(o Options, spec string) (action.Action, error) {
	prog := o.Program
	if prog == "" {
		return action.Action{}, errors.New("jobwatch does not know its own path, so it cannot schedule itself")
	}
	b := o.book()
	prev := ReadCheck(o)
	r := loom.Recipe{
		Name:     CheckRecipe,
		Plain:    "Alerts when a scheduled job has a problem.",
		Schedule: spec,
		Steps:    []loom.Step{{Label: "Check", Run: action.Pipeline{{Argv: checkArgv(prog, AlertsPath(o), prev.Shadow, prev.Notify)}}}},
	}
	if err := b.Save(r); err != nil {
		return action.Action{}, err
	}
	if loom.IsCron(spec) {
		return loom.CronAction(context.Background(), b.Exec, r, prog)
	}
	return loom.ScheduleAction(r, prog)
}

func scheduleOn(o Options, spec string, js bool) int {
	a, err := ScheduleCheck(o, spec)
	if err != nil {
		fmt.Fprintln(o.err(), err)
		return 1
	}
	if len(a.Undo) == 0 {
		fmt.Fprintln(o.err(), "this schedule has no undo")
		return 1
	}
	out := action.Doer{Exec: o.book().Exec, Ask: action.AskFunc(func(action.Plan, *action.Result) bool { return true })}.Do(context.Background(), a)
	if !out.OK() {
		if out.Err != nil {
			fmt.Fprintln(o.err(), out.Err)
		} else {
			fmt.Fprintln(o.err(), "the schedule was not turned on")
		}
		return 1
	}
	if js {
		return writeJSON(o, map[string]any{
			"name": CheckRecipe, "schedule": spec, "undo": true, "describe": a.Describe,
		})
	}
	fmt.Fprintln(o.out(), a.Describe)
	fmt.Fprintln(o.out(), "Scheduled. Undo puts the previous schedule back.")
	return 0
}

func scheduleUndo(o Options, js bool) int {
	b := o.book()
	a, err := UnscheduleCheck(o)
	if err != nil {
		fmt.Fprintln(o.err(), err)
		return 1
	}
	out := action.Doer{Exec: b.Exec, Ask: action.AskFunc(func(action.Plan, *action.Result) bool { return true })}.Do(context.Background(), a)
	if !out.OK() {
		if out.Err != nil {
			fmt.Fprintln(o.err(), out.Err)
		} else {
			fmt.Fprintln(o.err(), "the schedule was not turned off")
		}
		return 1
	}
	if js {
		return writeJSON(o, map[string]any{"name": CheckRecipe, "undo": true, "describe": a.Describe})
	}
	fmt.Fprintln(o.out(), a.Describe)
	return 0
}
