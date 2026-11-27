# Screenshots

The images the main README shows, made from the program itself and not drawn
by hand. They show seeded runs, not anyone's real jobs. These were taken from
a build of `main` at `7f594c8` with the fix for text wrapping beside the icon
rail.

1. Seed a history under a `HOME` made for the purpose (`/home/demo`), in
   `.local/share/mint-loom`, with `XDG_DATA_HOME` unset: the scheduled check
   is not passed that variable. Use `jobwatch import` and run records in
   Loom's format. Install jobwatch where a crontab would call it
   (`/usr/local/bin/jobwatch`), so the crontab change shows a real path. A
   stand-in `crontab` command keeps the real one untouched.
2. Turn the check on and set it to shadow. Run it once with
   `jobwatch --loom-run jobwatch-check` while every job is healthy, then add
   the failed run and use `jobwatch check --shadow` for the alerts. The order
   matters while a check that finds a problem is recorded as failed.
3. Open jobwatch once and quit, to get past the first-run quick start.
4. Run jobwatch in a pseudo-terminal 100 columns wide (52 for `phone.png`),
   and read the screen with a terminal emulator library such as pyte.
5. Draw each cell in DejaVu Sans Mono on `#14161a`, in its own colour, at
   twice the size (16 × 36 pixels a cell, with a 24-pixel margin). Block
   elements, `▓` and box lines are drawn as shapes, since a font's glyphs for
   them do not always fill the cell. A glyph DejaVu Sans Mono lacks comes
   from another font (`⏲` from FreeSerif).

| File | Screen | Size |
|---|---|---|
| `menu.png` | Menu, the first screen | 100 × 40 |
| `phone.png` | Menu | 52 × 42 |
| `overview.png` | Overview | 100 × 26 |
| `job.png` | a job (Enter on the first) | 100 × 44 |
| `day.png` | Day | 100 × 20 |
| `addjob.png` | Add job | 100 × 20 |
| `confirm.png` | Add job, a line not yet recorded, after naming it | 100 × 22 |
| `check.png` | Check | 100 × 20 |
| `guide.png` | Guide | 100 × 30 |
