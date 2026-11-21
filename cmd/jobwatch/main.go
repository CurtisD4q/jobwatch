// Command jobwatch is the app: are my backup and sync jobs still working,
// and will they keep working? It reads what mint-jw recorded, says each
// job's state in a word, draws the day, and shows the numbers behind every
// finding. It never changes a schedule.
package main

import (
	"os"
	"strings"

	mint "github.com/CurtisD4q/mint-kit"
	loom "github.com/CurtisD4q/mint-loom"
	"github.com/CurtisD4q/mint-loom/ui"
	"github.com/CurtisD4q/mint-tools/run"
)

// Version is the app's own version.
const Version = "0.1.0"

func main() {
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
	o := ui.Options{Book: book, Program: exe}
	app := &mint.App{
		Title:   "✦ Jobwatch",
		Version: Version,
		Group:   "Observe over time",
		Items:   ui.JobwatchEntries(o),
	}
	// With no arguments, start on the job list rather than the menu. --dir
	// is read above and left out of what Mint sees, so the two together
	// still count as no arguments.
	os.Args = append([]string{os.Args[0]}, flagRest(os.Args[1:])...)
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "--screen", "Jobs")
	}
	app.Run()
}

// flagDir is the --dir a person gave, and "" when there is none. It is read
// here rather than with the flag package so Mint's own flags keep working.
func flagDir(args []string) string {
	for i, a := range args {
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
