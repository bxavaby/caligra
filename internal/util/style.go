// BYZRA ⸻ internal/shared/style.go
// defines CLI visual style, color roles, ornaments, and motion

package util

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

type ColorConfig struct {
	Colors struct {
		CHRM string
		HEAT string
		HOTP string
		GUNM string
		VBLK string
		CSTL string
	}
}

// ╭─ COLOR ROLES ───────────────────────────────╮
var (
	CHRM lipgloss.Color
	HEAT lipgloss.Color
	HOTP lipgloss.Color
	GUNM lipgloss.Color
	VBLK lipgloss.Color
	CSTL lipgloss.Color
)

// ╭─ STYLE DEFINITIONS ─────────────────────────╮
var (
	BRH lipgloss.Style
	BRU lipgloss.Style
	LBL lipgloss.Style
	SUB lipgloss.Style
	NSH lipgloss.Style
	SHE lipgloss.Style
	SEC lipgloss.Style
	NLL lipgloss.Style
	ORN lipgloss.Style
)

func init() {
	// load from TOML
	config := loadColorConfig()

	CHRM = lipgloss.Color(config.Colors.CHRM)
	HEAT = lipgloss.Color(config.Colors.HEAT)
	HOTP = lipgloss.Color(config.Colors.HOTP)
	GUNM = lipgloss.Color(config.Colors.GUNM)
	VBLK = lipgloss.Color(config.Colors.VBLK)
	CSTL = lipgloss.Color(config.Colors.CSTL)

	BRH = lipgloss.NewStyle().Foreground(HOTP).Bold(true)
	BRU = lipgloss.NewStyle().Foreground(HOTP).Bold(true).Underline(true)
	LBL = lipgloss.NewStyle().Foreground(HEAT).Bold(true)
	SUB = lipgloss.NewStyle().Foreground(GUNM)
	NSH = lipgloss.NewStyle().Foreground(CHRM).Bold(true)
	SHE = lipgloss.NewStyle().Foreground(CHRM).Bold(true).Underline(true)
	SEC = lipgloss.NewStyle().Foreground(CSTL).Bold(true)
	NLL = lipgloss.NewStyle().Foreground(VBLK).Faint(true)
	ORN = lipgloss.NewStyle().Foreground(GUNM).Bold(true)
}

func loadColorConfig() ColorConfig {
	var config ColorConfig

	paths := []string{
		"yogra.toml",
		"data/yogra.toml",
		filepath.Join(os.Getenv("HOME"), "./caligra/config/yogra.toml"),
	}

	for _, path := range paths {
		if _, err := toml.DecodeFile(path, &config); err == nil {
			return config
		}
	}

	// panic or use defaults
	fmt.Println("Warning: Could not find yogra.toml, using hardcoded defaults")

	// default values
	config.Colors.CHRM = "#C0C0C0"
	config.Colors.HEAT = "#FF5C00"
	config.Colors.HOTP = "#FF007F"
	config.Colors.GUNM = "#444444"
	config.Colors.VBLK = "#121212"
	config.Colors.CSTL = "#88AABB"

	return config
}

// ╭─ ORNAMENT ──────────────────────────────────╮
var (
	Ornament = ORN.Render("›") // prefix UX lines
	Divider  = SUB.Render(strings.Repeat("─", 48))
)

// ╭─ SPINNER ───────────────────────────────────╮
func SpinWhile(label string, fn func() (string, error)) (string, error) {
	s := spinner.New(spinner.WithSpinner(spinner.Meter))
	ticker := time.NewTicker(s.Spinner.FPS)
	defer ticker.Stop()

	done := make(chan struct{})
	result := make(chan struct {
		out string
		err error
	})

	go func() {
		frame := 0
		frames := s.Spinner.Frames
		for {
			select {
			case <-ticker.C:
				fmt.Printf("\r%s %s", ORN.Render(frames[frame]), LBL.Render(label))
				frame = (frame + 1) % len(frames)
			case <-done:
				return
			}
		}
	}()

	go func() {
		out, err := fn()
		result <- struct {
			out string
			err error
		}{out, err}
	}()

	res := <-result
	close(done)
	Wiper()
	return res.out, res.err
}

func SuccessSymbol() string {
	return LBL.Render("[✓]")
}

func WarningSymbol() string {
	return SEC.Render("[!]")
}

func InfoSymbol() string {
	return NSH.Render("[i]")
}

func ErrorSymbol() string {
	return BRH.Render("[X]")
}

// ╭─ CLEAR ─────────────────────────────────────╮
func Wiper() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}
