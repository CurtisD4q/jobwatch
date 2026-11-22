package screens

import (
	"os"

	mint "github.com/CurtisD4q/mint-kit"
	"github.com/CurtisD4q/mint-kit/theme"
)

// jobwatchSpec is jobwatch's own look: night blue with sky accents, green
// for fine, and amber kept for warnings so that it always means something.
const jobwatchSpec = "accent=#7dd3fc,bright=#e0f2fe,text=#e2e8f0,muted=#94a3b8,on=#86efac,amber=#fbbf24,selbg=#1e3a5f,err=#fca5a5"

// UsePalette gives jobwatch its own colours, unless the person has chosen
// a theme (on the Theme screen or with MINT_PALETTE), which always wins.
func UsePalette() {
	if os.Getenv("MINT_PALETTE") != "" || theme.Current() != "default" {
		return
	}
	if p, err := theme.FromSpec(theme.Load("default"), jobwatchSpec); err == nil {
		mint.SetPalette(p)
	}
}
