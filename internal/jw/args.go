package jw

import (
	"flag"
	"io"
)

// split is args before and after the first "--". Nothing after it is ever
// read as a flag: it is a command and its own arguments.
func split(args []string) (before, after []string, dashed bool) {
	for i, a := range args {
		if a == "--" {
			return args[:i], args[i+1:], true
		}
	}
	return args, nil, false
}

// newFlags is a command's flags, reported by the caller, not printed.
func newFlags(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

// parse reads fs's flags from args, which may come before, between or
// after the positional words, and returns the words in order. args must
// not hold "--" (see split).
func parse(fs *flag.FlagSet, args []string) ([]string, error) {
	var words []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return words, nil
		}
		words = append(words, rest[0])
		args = rest[1:]
	}
}

// takeDir removes --dir PATH (or --dir=PATH) from the part of args before
// "--", wherever it is there, and returns it.
func takeDir(args []string) (string, []string) {
	before, after, dashed := split(args)
	dir := ""
	var kept []string
	for i := 0; i < len(before); i++ {
		a := before[i]
		switch {
		case a == "--dir" && i+1 < len(before):
			dir = before[i+1]
			i++
		case len(a) > 6 && a[:6] == "--dir=":
			dir = a[6:]
		default:
			kept = append(kept, a)
		}
	}
	if dashed {
		kept = append(append(kept, "--"), after...)
	}
	return dir, kept
}
