# jobwatch

An Emacs table of Loom's scheduled jobs.

`M-x jobwatch` runs `mint-jw report --json` and lists each job. `RET` opens that job's evidence. `p` draws the recent successful run lengths as an SVG where Emacs can show images, and shows the SVG source otherwise. `s` runs `mint-jw respace --json` and shows the plan. `t` opens the day: one line per job, when it usually starts, when its usual run ends, when its slowest recent run ends, the next job to start, and the gap before it — or that the slowest runs already reach it. A job's own line, such as "31% slower per GB", sits under it. Nothing in this file changes a schedule. Apply a move by editing the job's schedule yourself.

`M-x jobwatch-timeline` opens the day directly. It reads `mint-jw timeline --json` and the report, so a job opened from the day has the same evidence as the same job opened from the list, and the same state word. A job with fewer than three recent successful runs by its schedule is left out, which is the same rule the terminal picture uses. A weekly job is marked `(weekly)` and is never read as a daily one. A job known only from an outside log is marked lower confidence.

`jobwatch-program` is `mint-jw`. `jobwatch-window` is the re-spacing window, `2h` unless you set it.

Needs Emacs 27, and a `mint-jw` whose `report --json` includes `durations` (seconds of recent successful runs) and which has `timeline --json` (mint-loom pull request 23). Without `timeline`, the list, evidence, plot and re-spacing still work; only the day does not.

```
emacs -batch -l jobwatch.el -l jobwatch-test.el -f ert-run-tests-batch-and-exit
```

Apache-2.0.
