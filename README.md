# jobwatch

An Emacs table of Loom's scheduled jobs.

`M-x jobwatch` runs `mint-jw report --json` and lists each job. `RET` opens that job's evidence. `p` draws the recent successful run lengths as an SVG where Emacs can show images, and shows the SVG source otherwise. `s` runs `mint-jw respace --json` and shows the plan. Nothing in this file changes a schedule. Apply a move by editing the job's schedule yourself.

`jobwatch-program` is `mint-jw`. `jobwatch-window` is the re-spacing window, `2h` unless you set it.

Needs Emacs 27, and a `mint-jw` whose `report --json` includes `durations` (seconds of recent successful runs).

```
emacs -batch -l jobwatch.el -l jobwatch-test.el -f ert-run-tests-batch-and-exit
```

Apache-2.0.
