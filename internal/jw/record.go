package jw

import (
	"fmt"

	loom "github.com/CurtisD4q/pmin-loom"
)

func record(args []string, o Options) int {
	name, quiet, cmd, ok := splitRecord(args)
	if !ok {
		fmt.Fprintln(o.err(), "use: "+o.name()+" record NAME [--quiet] -- COMMAND...")
		fmt.Fprintln(o.err(), "   or: "+o.name()+" record --name NAME [--quiet] -- COMMAND...")
		return 2
	}
	// The loom flags are the same path a crontab line uses, so the exit
	// code, the overlap lock and the stored run stay in one place.
	loomArgs := []string{loom.RecordFlag, name}
	if quiet {
		loomArgs = append([]string{loom.RecordQuietFlag}, loomArgs...)
	}
	loomArgs = append(loomArgs, "--")
	loomArgs = append(loomArgs, cmd...)
	handled, code := loom.HandleArgs(o.book(), loomArgs)
	if !handled {
		return 2
	}
	return code
}

// splitRecord reads "NAME [--quiet] -- command" and "--name NAME [--quiet] -- command".
// The name is either the one positional word, or --name, not both.
func splitRecord(args []string) (name string, quiet bool, cmd []string, ok bool) {
	before, cmd, dashed := split(args)
	if !dashed || len(cmd) == 0 {
		return "", false, nil, false
	}
	fs := newFlags("record")
	nameFlag := fs.String("name", "", "")
	quietFlag := fs.Bool("quiet", false, "")
	words, err := parse(fs, before)
	if err != nil {
		return "", false, nil, false
	}
	switch {
	case *nameFlag != "" && len(words) == 0:
		name = *nameFlag
	case *nameFlag == "" && len(words) == 1:
		name = words[0]
	default:
		return "", false, nil, false
	}
	return name, *quietFlag, cmd, true
}
