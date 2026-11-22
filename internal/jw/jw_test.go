package jw

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CurtisD4q/mint-tools/action"
	"github.com/CurtisD4q/mint-tools/run"

	loom "github.com/CurtisD4q/mint-loom"
)

func opts(dir string) Options {
	return Options{Dir: dir, Exec: run.Local{}, Program: "/usr/local/bin/jobwatch", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
}

func buffers(o Options) (*bytes.Buffer, *bytes.Buffer) {
	return o.Stdout.(*bytes.Buffer), o.Stderr.(*bytes.Buffer)
}

func TestRecordReportAndCheck(t *testing.T) {
	dir := t.TempDir()
	o := opts(dir)
	if code := Main([]string{"record", "backup", "--", "false"}, o); code != 1 {
		t.Fatalf("record exit %d", code)
	}
	o = opts(dir)
	if code := Main([]string{"report"}, o); code != 1 {
		t.Fatalf("report exit %d", code)
	}
	out, _ := buffers(o)
	if !strings.Contains(out.String(), "backup") || !strings.Contains(out.String(), "Its last run failed.") {
		t.Fatalf("%s", out.String())
	}

	o = opts(dir)
	if code := Main([]string{"report", "--json"}, o); code != 1 {
		t.Fatal(code)
	}
	out, _ = buffers(o)
	var rep struct {
		Jobs []struct {
			Name  string `json:"name"`
			State string `json:"state"`
			Says  string `json:"says"`
		} `json:"jobs"`
		Findings []struct {
			Job  string `json:"job"`
			Kind string `json:"kind"`
			Says string `json:"says"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(out.Bytes(), &rep); err != nil {
		t.Fatal(err)
	}
	if len(rep.Jobs) != 1 || rep.Jobs[0].Name != "backup" || rep.Jobs[0].State != "failing" || rep.Jobs[0].Says != "Its last run failed." {
		t.Fatalf("%+v", rep.Jobs)
	}
	if len(rep.Findings) == 0 || rep.Findings[0].Kind != "failing" || rep.Findings[0].Says != "Its last run failed." {
		t.Fatalf("%+v", rep.Findings)
	}

	alert := filepathJoin(dir, "alerts")
	notified := 0
	o = opts(dir)
	o.Notify = func(text string) error {
		notified++
		if strings.Contains(text, "false") || strings.Contains(text, "command") {
			t.Fatalf("alert carried more than the finding: %s", text)
		}
		return nil
	}
	if code := Main([]string{"check", "--shadow", "--notify", "--file", alert}, o); code != 1 {
		t.Fatal(code)
	}
	if notified != 0 {
		t.Fatal("shadow notified")
	}
	if _, err := os.Stat(alert); !os.IsNotExist(err) {
		t.Fatal("shadow wrote the alert file")
	}
	shadow, err := os.ReadFile(filepathJoin(dir, "health", "shadow.log"))
	if err != nil || !strings.Contains(string(shadow), "backup: Its last run failed.") {
		t.Fatalf("%s %v", shadow, err)
	}

	o = opts(dir)
	o.Notify = func(text string) error {
		notified++
		return nil
	}
	if code := Main([]string{"check", "--notify", "--file", alert}, o); code != 1 {
		t.Fatal(code)
	}
	if notified != 1 {
		t.Fatalf("notify %d", notified)
	}
	got, err := os.ReadFile(alert)
	if err != nil || strings.TrimSpace(string(got)) != "backup: Its last run failed." {
		t.Fatalf("%s %v", got, err)
	}
}

func TestShadowKeepsItsOwnStreak(t *testing.T) {
	dir := t.TempDir()
	when := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	if err := os.MkdirAll(filepathJoin(dir, "runs"), 0o700); err != nil {
		t.Fatal(err)
	}
	// A day of runs before each check, twelve hours apart: one that
	// finished, and one skipped because it was still going. A warning only
	// holds again on a run the last check did not have, so each check
	// needs a new day.
	day := func(now time.Time) {
		ok := now.Add(-14 * time.Hour)
		skip := now.Add(-2 * time.Hour)
		appendRuns(t, dir, "nightly",
			loom.Run{Start: ok, End: ok.Add(time.Hour), By: "schedule", OK: true},
			loom.Run{Start: skip, End: skip, By: "schedule", Skip: "it is already running"})
	}
	logPath := filepathJoin(dir, "health", "shadow.log")
	var lines []string
	for i := 1; i <= 4; i++ {
		now := when.AddDate(0, 0, i-3) // the third check is at when
		day(now)
		o := opts(dir)
		o.Now = func() time.Time { return now }
		if code := Main([]string{"check", "--shadow"}, o); code != 0 {
			out, errb := buffers(o)
			t.Fatalf("shadow %d exit %d\n%s\n%s", i, code, out.String(), errb.String())
		}
		b, err := os.ReadFile(logPath)
		if i < 3 {
			if err == nil && strings.TrimSpace(string(b)) != "" {
				t.Fatalf("shadow check %d logged %q", i, b)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		lines = strings.FieldsFunc(string(b), func(r rune) bool { return r == '\n' })
		if len(lines) != 1 {
			t.Fatalf("after %d checks: %q", i, b)
		}
	}
	if !strings.HasPrefix(lines[0], "2026-11-20T12:00:00Z nightly: ") || !strings.Contains(lines[0], "still going") {
		t.Fatalf("line %q", lines[0])
	}
	if _, err := os.Stat(filepathJoin(dir, "health", "state.json")); !os.IsNotExist(err) {
		t.Fatal("shadow wrote the real state")
	}
	o := opts(dir)
	last := when.AddDate(0, 0, 1)
	o.Now = func() time.Time { return last }
	if code := Main([]string{"check"}, o); code != 0 {
		t.Fatal(code)
	}
	out, _ := buffers(o)
	if strings.Contains(out.String(), "still going") {
		t.Fatalf("real check sent the warning shadow had already counted:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "Nothing to alert on.") {
		t.Fatalf("%s", out.String())
	}
}

func TestForecastJSON(t *testing.T) {
	dir := t.TempDir()
	o := opts(dir)
	if code := Main([]string{"forecast", "--json"}, o); code != 0 {
		t.Fatal(code)
	}
	out, _ := buffers(o)
	if !strings.Contains(out.String(), `"findings": []`) {
		t.Fatalf("%s", out.String())
	}
}

func TestScheduleHasUndo(t *testing.T) {
	dir := t.TempDir()
	f := &crontab{}
	o := opts(dir)
	o.Exec = f
	if code := Main([]string{"schedule", "0 3 * * *", "--on"}, o); code != 0 {
		err, _ := buffers(o)
		t.Fatalf("exit %d\n%s", code, err.String())
	}
	out, _ := buffers(o)
	if !strings.Contains(out.String(), "Undo") || !strings.Contains(f.tab, "--loom-run jobwatch-check") {
		t.Fatalf("out:\n%s\ntab:\n%s", out.String(), f.tab)
	}
	r, err := o.book().Load(CheckRecipe)
	if err != nil || r.Steps[0].Run[0].Argv[1] != "check" {
		t.Fatalf("%v %+v", err, r)
	}
	o = opts(dir)
	o.Exec = f
	if code := Main([]string{"schedule", "--undo"}, o); code != 0 {
		_, errb := buffers(o)
		t.Fatal(errb.String())
	}
	if strings.Contains(f.tab, "jobwatch-check") {
		t.Fatalf("undo left the entry:\n%s", f.tab)
	}
}

func TestRecordNameAndQuiet(t *testing.T) {
	dir := t.TempDir()
	o := opts(dir)
	if code := Main([]string{"record", "--name", "sync", "--", "echo", "named-ok"}, o); code != 0 {
		_, errb := buffers(o)
		t.Fatalf("exit %d\n%s", code, errb.String())
	}
	if _, err := os.Stat(filepathJoin(dir, "runs", "sync.jsonl")); err != nil {
		t.Fatal(err)
	}
	o = opts(dir)
	if code := Main([]string{"record", "plain", "--", "true"}, o); code != 0 {
		_, errb := buffers(o)
		t.Fatalf("positional exit %d\n%s", code, errb.String())
	}
	markerDir := t.TempDir()
	const marker = "quiet-marker-zz"
	if err := os.WriteFile(filepathJoin(markerDir, marker), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	o = opts(dir)
	printed := captureStdout(t, func() {
		if code := Main([]string{"record", "--quiet", "--name", "hush", "--", "ls", markerDir}, o); code != 0 {
			_, errb := buffers(o)
			t.Fatalf("quiet exit %d\n%s", code, errb.String())
		}
	})
	if !strings.Contains(printed, marker) {
		t.Fatalf("quiet mode hid the output from the caller:\n%s", printed)
	}
	raw, err := os.ReadFile(filepathJoin(dir, "runs", "hush.jsonl"))
	if err != nil || strings.Contains(string(raw), marker) {
		t.Fatalf("stored output: %v\n%s", err, raw)
	}
	o = opts(dir)
	if code := Main([]string{"record", "also", "--quiet", "--", "ls", markerDir}, o); code != 0 {
		_, errb := buffers(o)
		t.Fatalf("positional quiet exit %d\n%s", code, errb.String())
	}
	raw, err = os.ReadFile(filepathJoin(dir, "runs", "also.jsonl"))
	if err != nil || strings.Contains(string(raw), marker) {
		t.Fatalf("positional quiet stored output: %v\n%s", err, raw)
	}
	o = opts(dir)
	if code := Main([]string{"record", "--name"}, o); code != 2 {
		t.Fatalf("usage exit %d", code)
	}
}

func TestImportedJobInReportAndCheck(t *testing.T) {
	dir := t.TempDir()
	b := (&Options{Dir: dir}).book()
	var runs []loom.Run
	for i := 0; i < 6; i++ {
		s := time.Date(2026, 10, 1+i, 2, 0, 0, 0, time.UTC)
		runs = append(runs, loom.Run{
			Start: s, End: s.Add(time.Minute), Elapsed: 60,
			OK: true, By: "import", Confidence: "low",
		})
	}
	fail := time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
	runs = append(runs, loom.Run{
		Start: fail, End: fail.Add(time.Minute), Elapsed: 60,
		OK: false, By: "import", Confidence: "low",
	})
	if err := b.ReplaceImport("nightly", runs); err != nil {
		t.Fatal(err)
	}
	when := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	o := opts(dir)
	o.Now = func() time.Time { return when }
	if code := Main([]string{"report"}, o); code != 1 {
		out, errb := buffers(o)
		t.Fatalf("report exit %d\n%s\n%s", code, out.String(), errb.String())
	}
	out, _ := buffers(o)
	if !strings.Contains(out.String(), "nightly") || !strings.Contains(out.String(), "lower confidence") || !strings.Contains(out.String(), "not finished successfully") {
		t.Fatalf("%s", out.String())
	}
	o = opts(dir)
	o.Now = func() time.Time { return when }
	if code := Main([]string{"report", "--json"}, o); code != 1 {
		t.Fatal(code)
	}
	out, _ = buffers(o)
	if !strings.Contains(out.String(), `"confidence": "low"`) {
		t.Fatalf("%s", out.String())
	}
	o = opts(dir)
	o.Now = func() time.Time { return when }
	if code := Main([]string{"check"}, o); code != 1 {
		out, errb := buffers(o)
		t.Fatalf("check exit %d\n%s\n%s", code, out.String(), errb.String())
	}
	out, _ = buffers(o)
	if !strings.Contains(out.String(), "nightly:") || !strings.Contains(out.String(), "not finished successfully") || !strings.Contains(out.String(), "(lower confidence)") {
		t.Fatalf("%s", out.String())
	}
}

func TestUsage(t *testing.T) {
	o := opts(t.TempDir())
	if code := Main(nil, o); code != 2 {
		t.Fatal(code)
	}
	if code := Main([]string{"record"}, o); code != 2 {
		t.Fatal(code)
	}
	if code := Main([]string{"nope"}, o); code != 2 {
		t.Fatal(code)
	}
}

func filepathJoin(elem ...string) string {
	return strings.Join(elem, "/")
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	w.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// crontab is a stand-in crontab command.
type crontab struct{ tab string }

func (c *crontab) Run(_ context.Context, p action.Pipeline) action.Result {
	if len(p) == 0 {
		return action.Result{Err: os.ErrInvalid}
	}
	argv := p[0].Argv
	if len(argv) >= 2 && argv[0] == "crontab" && argv[1] == "-l" {
		if c.tab == "" {
			return action.Result{Exit: 1}
		}
		return action.Result{Stdout: []byte(c.tab)}
	}
	if len(argv) >= 2 && argv[0] == "crontab" && argv[1] == "-" {
		c.tab = string(p[0].Stdin)
		return action.Result{}
	}
	if len(argv) >= 1 && argv[0] == "crontab" && argv[1] == "-r" {
		c.tab = ""
		return action.Result{}
	}
	return action.Result{}
}

var _ = loom.RunFlag

// A front end that draws its own 24-hour picture gets the numbers, and the
// state of each job comes from the same report, so the same job reads the
// same on the picture and in the list.
func TestTimelineJSON(t *testing.T) {
	dir := t.TempDir()
	b := (&Options{Dir: dir}).book()
	add := func(name string, hour int, length time.Duration) {
		var runs []loom.Run
		for i := 0; i < 8; i++ {
			s := time.Date(2026, 11, 2+i, hour, 0, 0, 0, time.UTC)
			runs = append(runs, loom.Run{
				Start: s, End: s.Add(length), Elapsed: length.Seconds(),
				OK: true, By: "import", Confidence: "low",
			})
		}
		if err := b.ReplaceImport(name, runs); err != nil {
			t.Fatal(err)
		}
	}
	add("backup", 2, 30*time.Minute)
	add("prune", 3, 10*time.Minute)
	when := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

	type bar struct {
		Name          string  `json:"name"`
		State         string  `json:"state"`
		Period        float64 `json:"period"`
		Start         float64 `json:"start"`
		Usual         float64 `json:"usual"`
		Slow          float64 `json:"slow"`
		Next          float64 `json:"next"`
		NextName      string  `json:"nextName"`
		Gap           float64 `json:"gap"`
		Overrun       bool    `json:"overrun"`
		LowConfidence bool    `json:"lowConfidence"`
	}

	o := opts(dir)
	o.Now = func() time.Time { return when }
	if code := Main([]string{"timeline", "--json"}, o); code != 0 {
		out, errb := buffers(o)
		t.Fatalf("timeline exit %d\n%s\n%s", code, out.String(), errb.String())
	}
	out, _ := buffers(o)
	var tl struct {
		Clock string `json:"clock"`
		Bars  []bar  `json:"bars"`
	}
	if err := json.Unmarshal(out.Bytes(), &tl); err != nil {
		t.Fatal(err)
	}
	if len(tl.Bars) != 2 || tl.Clock == "" {
		t.Fatalf("%s", out.String())
	}
	var backup *bar
	for i := range tl.Bars {
		if tl.Bars[i].Name == "backup" {
			backup = &tl.Bars[i]
		}
	}
	if backup == nil {
		t.Fatalf("%s", out.String())
	}
	if backup.Start != 120 || backup.NextName != "prune" || math.Abs(backup.Next-180) > 1 {
		t.Fatalf("%+v", backup)
	}
	if backup.Overrun || backup.Gap <= 0 {
		t.Fatalf("%+v", backup)
	}
	if backup.State == "" {
		t.Fatalf("no state: %+v", backup)
	}
	if !backup.LowConfidence {
		t.Fatalf("an imported job should be marked: %+v", backup)
	}

	// The text form is the picture, and it names both jobs.
	o = opts(dir)
	o.Now = func() time.Time { return when }
	if code := Main([]string{"timeline"}, o); code != 0 {
		t.Fatal(code)
	}
	out, _ = buffers(o)
	if !strings.Contains(out.String(), "backup") || !strings.Contains(out.String(), "prune") || !strings.Contains(out.String(), "#") {
		t.Fatalf("%s", out.String())
	}
	// Nothing recorded yet is an empty picture, not an error.
	o = opts(t.TempDir())
	if code := Main([]string{"timeline", "--json"}, o); code != 0 {
		t.Fatal(code)
	}
	out, _ = buffers(o)
	if !strings.Contains(out.String(), `"bars": []`) {
		t.Fatalf("%s", out.String())
	}
	// A flag it does not know is a usage error, as everywhere else.
	o = opts(dir)
	if code := Main([]string{"timeline", "--nope"}, o); code != 2 {
		t.Fatal(code)
	}
}

// A program that carries these commands says its own name in the help and
// in a usage error, so a person types what they have.
func TestOwnName(t *testing.T) {
	o := opts(t.TempDir())
	out, errb := buffers(o)
	if code := Main([]string{"--help"}, o); code != 0 {
		t.Fatalf("help exit %d", code)
	}
	if !strings.Contains(out.String(), "jobwatch record NAME") || strings.Contains(out.String(), "mint-jw") {
		t.Fatalf("help:\n%s", out)
	}
	if code := Main([]string{"record"}, o); code != 2 {
		t.Fatalf("usage exit %d", code)
	}
	if !strings.Contains(errb.String(), "use: jobwatch record") {
		t.Fatalf("usage error: %s", errb)
	}
	// a given name is used, as for a copy installed under another name
	o2 := opts(t.TempDir())
	o2.Name = "jw"
	out2, _ := buffers(o2)
	Main([]string{"help"}, o2)
	if !strings.Contains(out2.String(), "jw record NAME") {
		t.Fatalf("given name:\n%s", out2)
	}
}

// appendRuns adds runs to a job's history file, as Loom writes them.
func appendRuns(t *testing.T, dir, name string, runs ...loom.Run) {
	t.Helper()
	if err := os.MkdirAll(filepathJoin(dir, "runs"), 0o700); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(filepathJoin(dir, "runs", name+".jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, r := range runs {
		r.Recipe = name
		b, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		f.Write(append(b, '\n'))
	}
}

// A history opened with --dir is not the one cron lines record into, so
// jobwatch will not schedule the check from it.
func TestNoScheduleFromElsewhere(t *testing.T) {
	o := Options{Dir: t.TempDir(), Home: t.TempDir(), Program: "/usr/local/bin/jobwatch", Exec: run.Local{}}
	if _, err := ScheduleCheck(o, "0 * * * *"); err == nil || !strings.Contains(err.Error(), "--dir") {
		t.Fatalf("scheduled from another folder: %v", err)
	}
	if Elsewhere(o.Dir, o.Dir+"/") != "" || Elsewhere(o.Dir, "") != "" {
		t.Fatal("the same folder counted as elsewhere")
	}
}
