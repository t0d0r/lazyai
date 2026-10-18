package agent

import (
	"testing"
	"time"
)

func TestParsePSLinux(t *testing.T) {
	out := []byte(`    9     1  1001  0.0  4036 ?        Ss   Sun Oct 18 12:17:16 2026 tmux new-session -d -s x -- sleep 300
   10     9  1001  0.0  1280 pts/0    Ss+  Sun Oct  4 12:17:16 2026 sleep 300
`)
	procs, err := ParsePS(out)
	if err != nil || len(procs) != 2 {
		t.Fatalf("got %d procs, %v", len(procs), err)
	}
	if procs[0].TTY != "" || procs[1].TTY != "pts/0" {
		t.Errorf("tty: %q %q", procs[0].TTY, procs[1].TTY)
	}
}

func TestParseStat(t *testing.T) {
	// The command name holds a space and a parenthesis.
	line := []byte("4242 (my (agent) x) S 1 4242 4242 34816 4242 4194560 1210 0 0 0 150 50 0 0 20 0 1 0 98765 1234567 890 18446744073709551615\n")
	got, ok := parseStat(line)
	if !ok || got.used != 200 || got.start != 98765 {
		t.Errorf("parseStat = %+v, %v", got, ok)
	}
	if _, ok := parseStat([]byte("4242 (short) S 1 2")); ok {
		t.Error("short line parsed")
	}
}

func TestApplyCPU(t *testing.T) {
	procs := []Proc{{PID: 1, CPU: 0.1}, {PID: 2, CPU: 0.2}, {PID: 3, CPU: 0.3}, {PID: 4, CPU: 0.4}}
	prev := map[int]cpuTimes{1: {used: 100, start: 7}, 2: {used: 100, start: 7}, 3: {used: 0, start: 7}}
	cur := map[int]cpuTimes{1: {used: 250, start: 7}, 2: {used: 100, start: 7}, 3: {used: 50, start: 8}, 4: {used: 9, start: 7}}
	applyCPU(procs, prev, cur, 2*time.Second)
	// 150 ticks in 2 s is 0.75 s of CPU per second.
	want := []float64{75, 0, 0.3, 0.4} // 3 is a reused PID, 4 is new: ps values stay
	for i, p := range procs {
		if p.CPU != want[i] {
			t.Errorf("PID %d CPU %v, want %v", p.PID, p.CPU, want[i])
		}
	}
}
