// Command jobwatch is the app: are my backup and sync jobs still working,
// and will they keep working? With no command it opens the job list: each
// job's state in a word, the day drawn, and the numbers behind every
// finding. With a command (record, check, report, ...) it does that one
// thing and exits, so one program both records the jobs and shows them.
// It never changes a schedule.
package main

import (
	"os"
	"strings"

	"github.com/CurtisD4q/pmin-base/selfinstall"

	"github.com/CurtisD4q/jobwatch/internal/jw"
	"github.com/CurtisD4q/jobwatch/internal/screens"
	mint "github.com/CurtisD4q/pmin-kit"
	loom "github.com/CurtisD4q/pmin-loom"
	"github.com/CurtisD4q/pmin-tools/run"
)

// Version is the app's own version. A release sets it from its tag with
// -ldflags "-X main.Version=…"; a plain build says what it is from.
var Version = "0.1.0-dev"

func main() {
	// `jobwatch install` puts this program on the PATH, with jobwatch.el
	// beside it; `jobwatch uninstall` takes them off again.
	if ok, code := selfinstall.Handle(selfinstall.Program{Name: "jobwatch", Files: []string{"jobwatch.el"},
		After: "For Emacs, add to your init file: (load \"{data}/jobwatch.el\")"}, os.Args[1:], os.Stdout, os.Stderr); ok {
		os.Exit(code)
	}
	exe, _ := os.Executable()
	book := &loom.Book{Dir: loom.DefaultDir(), Exec: run.Local{}}
	// A schedule Loom made runs this program again with --loom-run, so a
	// jobwatch check can be scheduled from the same history.
	if handled, code := loom.HandleArgs(book, os.Args[1:]); handled {
		os.Exit(code)
	}
	// --dir reads a history kept somewhere else, which is how the app is
	// tried against a copy of real runs before it is pointed at the real
	// ones.
	if dir := flagDir(os.Args[1:]); dir != "" {
		book.Dir = dir
	}
	// A command such as `jobwatch record NAME -- restic backup /data` runs
	// and exits. jw reads --dir itself.
	if isCommand(flagRest(os.Args[1:])) {
		os.Exit(jw.Main(os.Args[1:], jw.Options{Dir: loom.DefaultDir(), Exec: run.Local{}, Program: exe, Name: "jobwatch", Home: loom.DefaultDir()}))
	}
	o := screens.Options{Book: book, Program: exe, Home: loom.DefaultDir(), Version: Version}
	screens.UsePalette()
	app := &mint.App{
		Title:   "✦ Jobwatch",
		Version: Version,
		Group:   "Observe over time",
		Items:   screens.Entries(o),
		Note:    screens.HomeNote(o),
		Status:  screens.HomeStatus(o),
		QuickStart: []string{
			"jobwatch watches the jobs cron runs. Put this in front of a",
			"crontab line, and each run of it is recorded:",
			"    jobwatch record NAME -- COMMAND",
			"",
			"Pick a screen from the menu; Overview shows every job, worst first.",
			"",
			"Tab or 1-3 moves between panes · / filters a list",
			"? explains how it works · q quits",
		},
	}
	os.Args = openScreen(os.Args)
	app.Run()
}

// commands are the jobs jobwatch does without opening the screens.
// `jobwatch help` lists them; --help is the screens' own help.
var commands = map[string]bool{
	"record": true, "report": true, "timeline": true, "forecast": true,
	"schedule": true, "check": true, "import": true, "respace": true,
	"help": true,
}

// isCommand is true when args (without --dir) start with a command.
func isCommand(args []string) bool {
	return len(args) > 0 && commands[args[0]]
}

// flagDir is the --dir a person gave, and "" when there is none. It is read
// here rather than with the flag package so Mint's own flags keep working.
func flagDir(args []string) string {
	for i, a := range args {
		if a == "--" {
			break // what follows is a command's own arguments
		}
		if a == "--dir" && i+1 < len(args) {
			return args[i+1]
		}
		if v, ok := strings.CutPrefix(a, "--dir="); ok {
			return v
		}
	}
	return ""
}

// flagRest is args without --dir and its value, so Mint's own flags read
// what is left.
func flagRest(args []string) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			return append(out, args[i:]...) // a command's own, untouched
		}
		if a == "--dir" {
			i++
			continue
		}
		if strings.HasPrefix(a, "--dir=") {
			continue
		}
		out = append(out, a)
	}
	return out
}

// openScreen is the arguments Mint sees, with the menu asked for when
// nothing was asked for. --dir is ours and left out, so `jobwatch --dir X`
// still counts as no arguments.
func openScreen(args []string) []string {
	// With no screen asked for, jobwatch opens on its launch menu.
	return append([]string{args[0]}, flagRest(args[1:])...)
}
