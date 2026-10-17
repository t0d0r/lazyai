// Package display formats agent fields for the terminal.
package display

import (
	"fmt"
	"os"
	"strings"
	"time"

	"agentctl/internal/agent"
)

// Bytes renders a size as "512K", "463M" or "1.2G".
func Bytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1fG", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%dM", n>>20)
	default:
		return fmt.Sprintf("%dK", n>>10)
	}
}

// Duration renders a span as "45s", "12m", "3h05m" or "2d04h".
func Duration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	default:
		return fmt.Sprintf("%dd%02dh", int(d.Hours())/24, int(d.Hours())%24)
	}
}

// Path shortens the home directory to "~".
func Path(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if p == home {
		return "~"
	}
	if rest, ok := strings.CutPrefix(p, home+"/"); ok {
		return "~/" + rest
	}
	return p
}

// Where is the short location column: session name and directory.
func Where(a agent.Agent) string {
	dir := Path(a.CWD)
	if dir == "" {
		dir = "-"
	}
	if a.Name != "" {
		return a.Name + "  " + dir
	}
	return dir
}

// Truncate cuts s to at most n runes, marking the cut with "~".
func Truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n == 1 {
		return "~"
	}
	return string(r[:n-1]) + "~"
}
