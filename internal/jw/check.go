package jw

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CurtisD4q/mint-loom/health"
	"github.com/CurtisD4q/mint-tools/action"
)

func check(args []string, o Options) int {
	fs := newFlags("check")
	file := fs.String("file", "", "")
	logPath := fs.String("log", "", "")
	asJSON := fs.Bool("json", false, "")
	shadow := fs.Bool("shadow", false, "")
	notify := fs.Bool("notify", false, "")
	if words, err := parse(fs, args); err != nil || len(words) > 0 {
		fmt.Fprintln(o.err(), usage(o.name()))
		return 2
	}
	js := map[string]bool{"--json": *asJSON, "--shadow": *shadow, "--notify": *notify}
	_, rep, err := load(o)
	if err != nil {
		fmt.Fprintln(o.err(), err)
		return 1
	}
	tr := health.Tracker{Dir: o.book().Dir, Now: o.now}
	var alerts []health.Finding
	if js["--shadow"] {
		alerts, err = tr.ApplyShadow(rep.Findings)
	} else {
		alerts, err = tr.Apply(rep.Findings)
	}
	if err != nil {
		fmt.Fprintln(o.err(), err)
		return 1
	}
	low := map[string]bool{}
	for _, s := range rep.Statuses {
		if s.Confidence == "low" {
			low[s.Job] = true
		}
	}
	lines := alertLines(alerts, low)
	if js["--shadow"] {
		if err := appendLines(shadowPath(o), stampLines(o.now(), lines)); err != nil {
			fmt.Fprintln(o.err(), err)
			return 1
		}
		return exitOf(rep)
	}
	if *file != "" {
		if err := appendLines(*file, lines); err != nil {
			fmt.Fprintln(o.err(), err)
			return 1
		}
	}
	if *logPath != "" {
		// the same lines, each with the time in front, for a history
		if err := appendLines(*logPath, stampLines(o.now(), lines)); err != nil {
			fmt.Fprintln(o.err(), err)
			return 1
		}
	}
	if js["--notify"] {
		for _, l := range lines {
			if err := o.notify(l); err != nil {
				fmt.Fprintln(o.err(), err)
				return 1
			}
		}
	}
	if js["--json"] {
		if writeJSON(o, map[string]any{"alerts": alertJSON(alerts)}) != 0 {
			return 1
		}
		return exitOf(rep)
	}
	for _, l := range lines {
		fmt.Fprintln(o.out(), l)
	}
	if *file == "" && *logPath == "" && !js["--notify"] && !js["--json"] && len(lines) == 0 {
		fmt.Fprintln(o.out(), "Nothing to alert on.")
	}
	return exitOf(rep)
}

func (o Options) notify(text string) error {
	if o.Notify != nil {
		return o.Notify(text)
	}
	r := o.book().Exec.Run(context.Background(), action.Pipeline{{Argv: []string{"notify-send", "Jobs", text}, Timeout: 10 * time.Second}})
	if r.Err != nil {
		return r.Err
	}
	if r.Exit != 0 {
		return fmt.Errorf("notify-send: %s", strings.TrimSpace(string(r.Stderr)))
	}
	return nil
}

// stampLines prefixes each alert with a UTC timestamp. The rest of the
// line is the alert a real check would have sent.
func stampLines(now time.Time, lines []string) []string {
	if len(lines) == 0 {
		return nil
	}
	stamp := now.UTC().Format(time.RFC3339)
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = stamp + " " + l
	}
	return out
}

func alertLines(fs []health.Finding, low map[string]bool) []string {
	var out []string
	for _, f := range fs {
		line := f.Job + ": " + f.Says
		if low[f.Job] {
			line += " (lower confidence)"
		}
		out = append(out, line)
	}
	return out
}

func alertJSON(fs []health.Finding) []findingJSON {
	var out []findingJSON
	for _, f := range fs {
		out = append(out, findingJSON{Job: f.Job, Kind: string(f.Kind), Level: f.Level.String(), Says: f.Says})
	}
	if out == nil {
		out = []findingJSON{}
	}
	return out
}

func shadowPath(o Options) string {
	return filepath.Join(o.book().Dir, "health", "shadow.log")
}

func appendLines(path string, lines []string) error {
	if len(lines) == 0 {
		return nil
	}
	if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, l := range lines {
		if _, err := fmt.Fprintln(f, l); err != nil {
			return err
		}
	}
	return nil
}

func dirOf(path string) string {
	i := strings.LastIndex(path, "/")
	if i <= 0 {
		return "."
	}
	return path[:i]
}
