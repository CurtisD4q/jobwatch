package jw

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/CurtisD4q/mint-tools/action"

	loom "github.com/CurtisD4q/mint-loom"
)

// ImportJournal reads journalctl for one unit and stores the runs it can
// see: start, finish, exit status and a size when a line states one.
// The messages themselves are not stored.
func ImportJournal(o Options, unit, since string) (int, error) {
	return ImportJournalFrom(o, unit, since, false)
}

// journalSince is since as journalctl reads it. A span such as "90d" is
// how far back to go, which journalctl writes "-90d"; it refuses "90d"
// itself ("Failed to parse timestamp"). A date or "yesterday" passes on.
func journalSince(since string) string {
	since = strings.TrimSpace(since)
	if since == "" {
		since = "90d"
	}
	if since[0] >= '0' && since[0] <= '9' && !strings.ContainsAny(since, "-:") {
		return "-" + since
	}
	return since
}

// ImportJournalFrom is ImportJournal, reading the person's own journal
// (journalctl --user) when user is true: a user timer's service is there.
func ImportJournalFrom(o Options, unit, since string, user bool) (int, error) {
	unit = strings.TrimSpace(unit)
	if unit == "" {
		return 0, fmt.Errorf("give --unit, the service to read")
	}
	argv := []string{"journalctl"}
	if user {
		argv = append(argv, "--user")
	}
	argv = append(argv, "-u", unit, "--since="+journalSince(since), "--no-pager", "-o", "short-iso")
	r := o.book().Exec.Run(context.Background(), action.Pipeline{{Argv: argv, Timeout: 30 * time.Second}})
	if r.Err != nil {
		return 0, r.Err
	}
	if r.Exit != 0 && len(r.Stdout) == 0 {
		msg := strings.TrimSpace(string(r.Stderr))
		if msg == "" {
			msg = "journalctl could not read that unit"
		}
		return 0, fmt.Errorf("%s", msg)
	}
	name := strings.TrimSuffix(unit, ".service")
	runs := ParseJournal(string(r.Stdout), name)
	if err := o.book().ReplaceImport(name, runs); err != nil {
		return 0, err
	}
	return len(runs), nil
}

// ImportLog reads an rclone, rsync or cron log from text the caller already
// has. kind is "rclone", "rsync" or "cron".
func ImportLog(o Options, kind, name, text string, now time.Time) (int, error) {
	if strings.TrimSpace(name) == "" {
		return 0, fmt.Errorf("give --name, what to call the job")
	}
	var runs []loom.Run
	switch kind {
	case "rclone":
		runs = ParseRclone(text, name)
	case "rsync":
		runs = ParseRsync(text, name)
	case "cron":
		runs = ParseCron(text, name, now)
	default:
		return 0, fmt.Errorf("unknown import %q", kind)
	}
	if err := o.book().ReplaceImport(name, runs); err != nil {
		return 0, err
	}
	return len(runs), nil
}

func importCmd(args []string, o Options) int {
	if len(args) == 0 {
		fmt.Fprintln(o.err(), "use: "+o.name()+" import journal|rclone|rsync|cron")
		return 2
	}
	switch args[0] {
	case "journal":
		fs := newFlags("import journal")
		unitFlag := fs.String("unit", "", "")
		sinceFlag := fs.String("since", "", "")
		userFlag := fs.Bool("user", false, "")
		if words, err := parse(fs, args[1:]); err != nil || len(words) > 0 || *unitFlag == "" {
			fmt.Fprintln(o.err(), "use: "+o.name()+" import journal --unit UNIT [--since 90d] [--user]")
			return 2
		}
		unit, since := *unitFlag, *sinceFlag
		n, err := ImportJournalFrom(o, unit, since, *userFlag)
		if err != nil {
			fmt.Fprintln(o.err(), err)
			return 1
		}
		fmt.Fprintf(o.out(), "Imported %d run%s from %s.\n", n, plural(n), strings.TrimSuffix(unit, ".service"))
		return 0
	case "rclone", "rsync", "cron":
		fs := newFlags("import " + args[0])
		nameFlag := fs.String("name", "", "")
		paths, err := parse(fs, args[1:])
		name := *nameFlag
		if err != nil || len(paths) != 1 || name == "" {
			fmt.Fprintf(o.err(), "use: "+o.name()+" import %s --name JOB FILE\n", args[0])
			return 2
		}
		b, err := os.ReadFile(paths[0])
		if err != nil {
			fmt.Fprintln(o.err(), err)
			return 1
		}
		n, err := ImportLog(o, args[0], name, string(b), o.now())
		if err != nil {
			fmt.Fprintln(o.err(), err)
			return 1
		}
		fmt.Fprintf(o.out(), "Imported %d run%s of %s.\n", n, plural(n), name)
		return 0
	default:
		fmt.Fprintln(o.err(), "use: "+o.name()+" import journal|rclone|rsync|cron")
		return 2
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// The parsers below keep timings, an exit status and a size. They do not
// keep the line they read them from.

var journalLine = regexp.MustCompile(`^(\S+)\s+\S+\s+\S+:\s+(.*)$`)
var exitStatus = regexp.MustCompile(`status=(\d+)`)
var transferred = regexp.MustCompile(`(?i)transferred:\s*([0-9.]+\s*[KMGT]?i?B)`)

// ParseJournal builds runs from journalctl -o short-iso output.
func ParseJournal(text, name string) []loom.Run {
	var runs []loom.Run
	var cur *loom.Run
	var exitCode int
	var sawExit bool
	flush := func(end time.Time, ok bool) {
		if cur == nil {
			return
		}
		if !end.IsZero() {
			cur.End = end
			cur.Elapsed = end.Sub(cur.Start).Seconds()
			if cur.Elapsed < 0 {
				cur.Elapsed = 0
			}
			cur.OK = ok
		} else {
			cur.OK = true
		}
		runs = append(runs, *cur)
		cur = nil
		sawExit = false
	}
	for _, line := range strings.Split(text, "\n") {
		m := journalLine.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		when, ok := parseISO(m[1])
		if !ok {
			continue
		}
		msg := m[2]
		switch {
		case strings.HasPrefix(msg, "Started "):
			flush(time.Time{}, false)
			cur = &loom.Run{Recipe: name, Start: when, By: "import", Confidence: "low", OK: true}
			exitCode, sawExit = 0, false
		case strings.Contains(msg, "Main process exited"):
			if sm := exitStatus.FindStringSubmatch(msg); sm != nil {
				exitCode, _ = strconv.Atoi(sm[1])
				sawExit = true
			}
		case strings.Contains(msg, "Failed with result"):
			if cur != nil && sawExit && exitCode != 0 {
				cur.OK = false
			}
			flush(when, false)
		case strings.Contains(msg, "Deactivated successfully") || strings.HasPrefix(msg, "Finished "):
			ok := true
			if sawExit && exitCode != 0 {
				ok = false
			}
			flush(when, ok)
		default:
			if cur != nil && cur.Bytes == 0 {
				if sm := transferred.FindStringSubmatch(msg); sm != nil {
					if n, ok := parseSize(sm[1]); ok {
						cur.Bytes = n
					}
				}
			}
		}
	}
	flush(time.Time{}, false)
	return runs
}

func parseISO(s string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z0700", "2006-01-02T15:04:05-0700"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

var rcloneLine = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})\s+(.*)$`)
var elapsedLine = regexp.MustCompile(`(?i)^Elapsed time:\s+(\S+)`)

// ParseRclone reads an rclone log: a start, an elapsed time and a size.
func ParseRclone(text, name string) []loom.Run {
	var runs []loom.Run
	var start time.Time
	var size int64
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if sm := transferred.FindStringSubmatch(line); sm != nil {
			if n, ok := parseSize(sm[1]); ok {
				size = n
			}
		}
		if m := rcloneLine.FindStringSubmatch(line); m != nil {
			if t, err := time.ParseInLocation("2006/01/02 15:04:05", m[1], time.UTC); err == nil {
				if start.IsZero() {
					start = t
				}
			}
			continue
		}
		if m := elapsedLine.FindStringSubmatch(line); m != nil && !start.IsZero() {
			d, err := time.ParseDuration(m[1])
			if err != nil {
				d = 0
			}
			runs = append(runs, loom.Run{
				Recipe: name, Start: start, End: start.Add(d), Elapsed: d.Seconds(),
				OK: true, By: "import", Confidence: "low", Bytes: size,
			})
			start = time.Time{}
			size = 0
		}
	}
	return runs
}

var rsyncStamp = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})`)
var rsyncSent = regexp.MustCompile(`sent\s+([0-9,]+)\s+bytes`)

// ParseRsync reads an rsync log: the first and last timestamps are the
// length, and "sent N bytes" is the size.
func ParseRsync(text, name string) []loom.Run {
	var runs []loom.Run
	var start, last time.Time
	var size int64
	close := func() {
		if start.IsZero() {
			return
		}
		end := last
		if end.Before(start) {
			end = start
		}
		runs = append(runs, loom.Run{
			Recipe: name, Start: start, End: end, Elapsed: end.Sub(start).Seconds(),
			OK: true, By: "import", Confidence: "low", Bytes: size,
		})
		start, last, size = time.Time{}, time.Time{}, 0
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if m := rsyncStamp.FindStringSubmatch(line); m != nil {
			t, err := time.ParseInLocation("2006/01/02 15:04:05", m[1], time.UTC)
			if err == nil {
				if start.IsZero() {
					start = t
				}
				last = t
			}
		}
		if strings.HasPrefix(line, "building file list") || strings.Contains(line, "building file list") {
			if !start.IsZero() && !last.Equal(start) {
				close()
				if m := rsyncStamp.FindStringSubmatch(line); m != nil {
					t, _ := time.ParseInLocation("2006/01/02 15:04:05", m[1], time.UTC)
					start, last = t, t
				}
			}
		}
		if m := rsyncSent.FindStringSubmatch(line); m != nil {
			n, _ := strconv.ParseInt(strings.ReplaceAll(m[1], ",", ""), 10, 64)
			size = n
			close()
		}
	}
	return runs
}

var cronLine = regexp.MustCompile(`^([A-Z][a-z]{2})\s+(\d{1,2})\s+(\d{2}:\d{2}:\d{2})\s+\S+\s+CRON\[\d+\]:`)

// ParseCron reads cron log lines. Each is a start and nothing else, so it
// can show a missed run and cannot show how long the job took. The command
// in the line is not kept.
func ParseCron(text, name string, now time.Time) []loom.Run {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var runs []loom.Run
	for _, line := range strings.Split(text, "\n") {
		m := cronLine.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		when, ok := cronWhen(m[1], m[2], m[3], now)
		if !ok {
			continue
		}
		runs = append(runs, loom.Run{Recipe: name, Start: when, OK: true, By: "import", Confidence: "low"})
	}
	return runs
}

func cronWhen(mon, day, clock string, now time.Time) (time.Time, bool) {
	t, err := time.Parse("Jan 2 15:04:05", fmt.Sprintf("%s %s %s", mon, day, clock))
	if err != nil {
		return time.Time{}, false
	}
	when := time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.UTC)
	if when.After(now.Add(24 * time.Hour)) {
		when = when.AddDate(-1, 0, 0)
	}
	return when, true
}

func parseSize(s string) (int64, bool) {
	s = strings.ReplaceAll(strings.TrimSpace(s), " ", "")
	i := 0
	for i < len(s) && (s[i] == '.' || (s[i] >= '0' && s[i] <= '9')) {
		i++
	}
	if i == 0 {
		return 0, false
	}
	n, err := strconv.ParseFloat(s[:i], 64)
	if err != nil {
		return 0, false
	}
	unit := strings.ToUpper(s[i:])
	mult := 1.0
	switch unit {
	case "B", "":
		mult = 1
	case "KIB", "KB":
		mult = 1 << 10
	case "MIB", "MB":
		mult = 1 << 20
	case "GIB", "GB":
		mult = 1 << 30
	case "TIB", "TB":
		mult = 1 << 40
	default:
		return 0, false
	}
	return int64(n * mult), true
}
