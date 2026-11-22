package jw

import (
	"fmt"
	"strings"

	loom "github.com/CurtisD4q/mint-loom"
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
	var positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			cmd = args[i+1:]
			break
		}
		switch {
		case a == "--quiet":
			quiet = true
		case a == "--name" || strings.HasPrefix(a, "--name="):
			val := strings.TrimPrefix(a, "--name=")
			if a == "--name" {
				if i+1 >= len(args) || args[i+1] == "--" || strings.HasPrefix(args[i+1], "-") {
					return "", false, nil, false
				}
				i++
				val = args[i]
			}
			if val == "" || name != "" {
				return "", false, nil, false
			}
			name = val
		case strings.HasPrefix(a, "-"):
			return "", false, nil, false
		default:
			positional = append(positional, a)
		}
	}
	if len(cmd) == 0 {
		return "", false, nil, false
	}
	if name == "" {
		if len(positional) != 1 {
			return "", false, nil, false
		}
		name = positional[0]
	} else if len(positional) != 0 {
		return "", false, nil, false
	}
	return name, quiet, cmd, true
}
