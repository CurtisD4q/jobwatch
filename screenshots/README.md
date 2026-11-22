# Screenshots

The images the main README shows, made from the program itself and not drawn
by hand. They show seeded runs, not anyone's real jobs.

1. Seed a history in a folder of its own, with `XDG_DATA_HOME` pointing at
   its parent, so jobwatch reads it as the default (`jobwatch import`, or a
   few recorded runs), and install jobwatch where a crontab would call it
   (`/usr/local/bin/jobwatch`), so the crontab change shows a real path.
2. Run jobwatch in a pseudo-terminal 100 columns wide (52 for `phone.png`),
   and read the screen with a terminal emulator library such as pyte.
3. Draw each cell in DejaVu Sans Mono on `#14161a`, in its own colour, at
   twice the size. Block elements (`▁`…`█`, `▓`) are drawn as rectangles,
   since a font's glyphs for them do not always fill the cell.

| File | Screen | Size |
|---|---|---|
| `overview.png` | Overview | 100 × 26 |
| `phone.png` | Overview | 52 × 46 |
| `job.png` | a job (Enter on the first) | 100 × 40 |
| `day.png` | The day | 100 × 18 |
| `addjob.png` | Add a job | 100 × 18 |
| `confirm.png` | Add a job, Enter on a line | 100 × 22 |
| `check.png` | Background check | 100 × 18 |
| `guide.png` | How this works | 100 × 30 |
