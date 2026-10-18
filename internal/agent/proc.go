package agent

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Proc is one row of a ps snapshot.
type Proc struct {
	PID   int       `json:"pid"`
	PPID  int       `json:"ppid"`
	UID   int       `json:"uid"`
	CPU   float64   `json:"cpu"`
	RSS   int64     `json:"rss_bytes"`
	TTY   string    `json:"tty,omitempty"` // "ttys003" or "pts/3", empty when the process has no terminal
	Stat  string    `json:"stat"`
	Start time.Time `json:"started"`
	Args  string    `json:"-"` // raw command line; may hold secrets, see RedactArgs
}

// psFormat lists the ps columns in the order ParsePS expects. lstart is
// five whitespace-separated words; args is everything after it.
const psFormat = "pid=,ppid=,uid=,pcpu=,rss=,tty=,stat=,lstart=,args="

// lstartLayout matches ps lstart under LC_ALL=C after collapsing spaces.
const lstartLayout = "Mon Jan 2 15:04:05 2006"

// fixedFields is the number of words before args: 7 columns plus 5 for lstart.
const fixedFields = 12

// Snapshot returns every process visible to the current user. On Linux
// the CPU column is replaced with recent usage, see recentCPU.
func Snapshot() ([]Proc, error) {
	procs, err := runPS("-axww")
	if err != nil {
		return nil, err
	}
	recentCPU(procs)
	return procs, nil
}

// Lookup returns the current row for one PID, or ErrNoProcess.
func Lookup(pid int) (Proc, error) {
	procs, err := runPS("-ww", "-p", strconv.Itoa(pid))
	if err != nil || len(procs) == 0 {
		return Proc{}, ErrNoProcess
	}
	return procs[0], nil
}

// ErrNoProcess means the PID does not exist (any more).
var ErrNoProcess = errors.New("no such process")

func runPS(args ...string) ([]Proc, error) {
	cmd := exec.Command("ps", append(args, "-o", psFormat)...)
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("ps: %w", err)
	}
	return ParsePS(out)
}

// ParsePS parses output produced with psFormat. Malformed lines are skipped.
func ParsePS(out []byte) ([]Proc, error) {
	var procs []Proc
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		p, ok := parseLine(sc.Text())
		if ok {
			procs = append(procs, p)
		}
	}
	return procs, sc.Err()
}

func parseLine(line string) (Proc, bool) {
	fields, rest := splitFields(line, fixedFields)
	if len(fields) < fixedFields || rest == "" {
		return Proc{}, false
	}
	var p Proc
	var err error
	if p.PID, err = strconv.Atoi(fields[0]); err != nil {
		return Proc{}, false
	}
	if p.PPID, err = strconv.Atoi(fields[1]); err != nil {
		return Proc{}, false
	}
	if p.UID, err = strconv.Atoi(fields[2]); err != nil {
		return Proc{}, false
	}
	if p.CPU, err = strconv.ParseFloat(fields[3], 64); err != nil {
		return Proc{}, false
	}
	kb, err := strconv.ParseInt(fields[4], 10, 64)
	if err != nil {
		return Proc{}, false
	}
	p.RSS = kb * 1024
	if fields[5] != "??" && fields[5] != "?" && fields[5] != "-" { // macOS "??", Linux "?"
		p.TTY = fields[5]
	}
	p.Stat = fields[6]
	if p.Start, err = ParseLstart(strings.Join(fields[7:12], " ")); err != nil {
		return Proc{}, false
	}
	p.Args = rest
	return p, true
}

// ParseLstart parses a ps lstart value such as "Sat Oct  5 21:20:54 2026".
func ParseLstart(s string) (time.Time, error) {
	return time.ParseInLocation(lstartLayout, strings.Join(strings.Fields(s), " "), time.Local)
}

// splitFields returns the first n whitespace-separated words of s and the
// remainder with its inner spacing preserved.
func splitFields(s string, n int) ([]string, string) {
	fields := make([]string, 0, n)
	i := 0
	for len(fields) < n {
		for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
			i++
		}
		if i == len(s) {
			return fields, ""
		}
		j := i
		for j < len(s) && s[j] != ' ' && s[j] != '\t' {
			j++
		}
		fields = append(fields, s[i:j])
		i = j
	}
	return fields, strings.TrimSpace(s[i:])
}
