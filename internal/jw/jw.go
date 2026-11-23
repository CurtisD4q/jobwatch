// Package jw is jobwatch's commands: record a run, report how jobs are
// doing, forecast an overrun, schedule the check, and raise an alert.
// Alerts carry the finding only. They never carry a command or its output.
package jw

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	jobhealth "github.com/CurtisD4q/pmin-jobhealth"
	loom "github.com/CurtisD4q/pmin-loom"
	"github.com/CurtisD4q/pmin-tools/action"
	"github.com/CurtisD4q/pmin-tools/run"
)

// CheckRecipe is the recipe whose schedule runs the check.
const CheckRecipe = "jobwatch-check"

// Options is how the command reaches the machine. Tests set them.
type Options struct {
	Dir     string
	Exec    action.Executor
	Program string
	Now     func() time.Time
	Notify  func(text string) error
	Stdout  io.Writer
	Stderr  io.Writer
	// Name is what the command is called in help and messages: "jobwatch"
	// when empty.
	Name string
	// Home is the history a cron line records into: cron runs jobwatch
	// without --dir, so it is always the default folder. Empty is Dir
	// (tests, where the two are the same).
	Home string
}

// Elsewhere is why jobwatch will not write a schedule or a crontab line
// for this history, or "" when it may. A history opened with --dir is not
// the one cron lines record into, so a line written from it would fill
// another folder while this one stayed empty.
func Elsewhere(dir, home string) string {
	if dir == "" || home == "" || filepath.Clean(dir) == filepath.Clean(home) {
		return ""
	}
	return "this history was opened with --dir, and cron runs jobwatch without it, so a schedule made here would record into " + home + " instead. Open jobwatch without --dir to schedule"
}

func (o Options) name() string {
	if o.Name == "" {
		return "jobwatch"
	}
	return o.Name
}

func (o Options) book() *loom.Book {
	dir := o.Dir
	if dir == "" {
		dir = loom.DefaultDir()
	}
	exec := o.Exec
	if exec == nil {
		exec = run.Local{}
	}
	b := &loom.Book{Dir: dir, Exec: exec}
	if o.Now != nil {
		b.Now = o.Now
	}
	return b
}

func (o Options) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

func (o Options) out() io.Writer {
	if o.Stdout != nil {
		return o.Stdout
	}
	return os.Stdout
}

func (o Options) err() io.Writer {
	if o.Stderr != nil {
		return o.Stderr
	}
	return os.Stderr
}

// Main runs jobwatch's commands. Exit codes: 0 fine, 1 problems, 2 usage.
func Main(args []string, o Options) int {
	if len(args) == 0 {
		fmt.Fprintln(o.err(), usage(o.name()))
		return 2
	}
	dir, args := takeDir(args)
	if dir != "" {
		o.Dir = dir
	}
	switch args[0] {
	case "record":
		return record(args[1:], o)
	case "report":
		return report(args[1:], o)
	case "timeline":
		return timeline(args[1:], o)
	case "forecast":
		return forecast(args[1:], o)
	case "schedule":
		return schedule(args[1:], o)
	case "check":
		return check(args[1:], o)
	case "import":
		return importCmd(args[1:], o)
	case "respace":
		return respaceCmd(args[1:], o)
	case "-h", "--help", "help":
		fmt.Fprintln(o.out(), usage(o.name()))
		return 0
	default:
		fmt.Fprintln(o.err(), usage(o.name()))
		return 2
	}
}

// usage is the help, with name as the command.
func usage(name string) string {
	return strings.ReplaceAll(`{cmd} record NAME [--quiet] -- COMMAND...
{cmd} record --name NAME [--quiet] -- COMMAND...
{cmd} report [--json]
{cmd} timeline [--json]
{cmd} forecast [--json]
{cmd} schedule SPEC [--json] [--on | --undo]
{cmd} check [--json] [--shadow] [--file PATH] [--log PATH] [--notify]
{cmd} import journal --unit UNIT [--since 90d] [--user]
{cmd} import rclone|rsync|cron --name JOB FILE
{cmd} respace [--window 2h] [--json]

Exit 0 is fine, 1 is a problem, 2 is usage.
check writes one line per alert. An alert is the finding only.
--shadow writes what it would have alerted on and sends nothing.`, "{cmd}", name)
}

func exitOf(rep jobhealth.Report) int {
	for _, f := range rep.Findings {
		if f.Level >= jobhealth.Problem {
			return 1
		}
	}
	return 0
}

func writeJSON(o Options, v any) int {
	enc := json.NewEncoder(o.out())
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(o.err(), err)
		return 1
	}
	return 0
}

// jsonFlag reads a command whose only flag is --json, and no words.
func jsonFlag(args []string) (map[string]bool, bool) {
	fs := newFlags("json")
	asJSON := fs.Bool("json", false, "")
	if words, err := parse(fs, args); err != nil || len(words) > 0 {
		return nil, false
	}
	return map[string]bool{"--json": *asJSON}, true
}
