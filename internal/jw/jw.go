// Package jw is jobwatch's commands: record a run, report how jobs are
// doing, forecast an overrun, schedule the check, and raise an alert.
// Alerts carry the finding only. They never carry a command or its output.
package jw

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	loom "github.com/CurtisD4q/mint-loom"
	"github.com/CurtisD4q/mint-loom/health"
	"github.com/CurtisD4q/mint-tools/action"
	"github.com/CurtisD4q/mint-tools/run"
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
	dir, args := take(args, "--dir")
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
{cmd} import journal --unit UNIT [--since 90d]
{cmd} import rclone|rsync|cron --name JOB FILE
{cmd} respace [--window 2h] [--json]

Exit 0 is fine, 1 is a problem, 2 is usage.
check writes one line per alert. An alert is the finding only.
--shadow writes what it would have alerted on and sends nothing.`, "{cmd}", name)
}

func exitOf(rep health.Report) int {
	for _, f := range rep.Findings {
		if f.Level >= health.Problem {
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

// flags reports which of known were present. An unknown flag is bad.
func flags(args []string, known map[string]bool) (map[string]bool, bool) {
	got := map[string]bool{}
	for _, a := range args {
		if !strings.HasPrefix(a, "--") {
			continue
		}
		name := a
		if i := strings.IndexByte(a, '='); i >= 0 {
			name = a[:i]
		}
		if _, ok := known[name]; !ok {
			return nil, true
		}
		got[name] = true
	}
	return got, false
}

func remain(args []string) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "--") {
			if !strings.Contains(a, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") && takesValue(a) {
				i++
			}
			continue
		}
		out = append(out, a)
	}
	return out
}

func takesValue(flag string) bool {
	return flag == "--dir" || flag == "--file" || flag == "--log"
}

// take removes --name value or --name=value.
func take(args []string, name string) (string, []string) {
	var rest []string
	val := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == name && i+1 < len(args) {
			val = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(a, name+"=") {
			val = strings.TrimPrefix(a, name+"=")
			continue
		}
		rest = append(rest, a)
	}
	return val, rest
}
