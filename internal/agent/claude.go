package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// claudeSession is the subset of ~/.claude/sessions/<pid>.json we use.
// The neighbouring *.key files are never read.
type claudeSession struct {
	PID             int    `json:"pid"`
	SessionID       string `json:"sessionId"`
	CWD             string `json:"cwd"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	StatusUpdatedAt int64  `json:"statusUpdatedAt"`
	ProcStart       string `json:"procStart"`
}

// ClaudeSessionsDir returns where Claude Code writes per-process metadata.
func ClaudeSessionsDir() string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "sessions")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "sessions")
}

// loadClaudeSession reads the metadata file for p. It returns false when the
// file is missing or was left behind by an earlier process with the same PID.
func loadClaudeSession(dir string, p Proc) (claudeSession, bool) {
	if dir == "" {
		return claudeSession{}, false
	}
	data, err := os.ReadFile(filepath.Join(dir, strconv.Itoa(p.PID)+".json"))
	if err != nil {
		return claudeSession{}, false
	}
	var s claudeSession
	if json.Unmarshal(data, &s) != nil || s.PID != p.PID {
		return claudeSession{}, false
	}
	if s.ProcStart != "" && !sameStart(s.ProcStart, p.Start) {
		return claudeSession{}, false
	}
	return s, true
}

// sameStart compares a procStart string with a ps start time. Claude Code
// writes procStart in UTC, ps reports local time; accept either.
func sameStart(procStart string, start time.Time) bool {
	norm := strings.Join(strings.Fields(procStart), " ")
	for _, loc := range []*time.Location{time.UTC, time.Local} {
		if t, err := time.ParseInLocation(lstartLayout, norm, loc); err == nil && t.Equal(start) {
			return true
		}
	}
	return false
}

func (s claudeSession) statusSince() time.Time {
	if s.StatusUpdatedAt == 0 {
		return time.Time{}
	}
	return time.UnixMilli(s.StatusUpdatedAt)
}

// resumeID returns the session id from "claude --resume <id>" or "-r <id>".
func resumeID(args string) string {
	tokens := strings.Fields(args)
	for i, t := range tokens {
		if (t == "--resume" || t == "-r") && i+1 < len(tokens) {
			return tokens[i+1]
		}
		if v, ok := strings.CutPrefix(t, "--resume="); ok {
			return v
		}
	}
	return ""
}
