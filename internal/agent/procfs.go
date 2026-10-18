package agent

import (
	"bufio"
	"bytes"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// procRoot is where Linux describes its processes. On macOS it does not
// exist and the functions in this file change nothing.
const procRoot = "/proc"

// clockTicks is USER_HZ, the unit of the times in /proc/<pid>/stat. Linux
// fixes it at 100 for user space whatever the kernel's own tick rate.
const clockTicks = 100

// firstSample is how long the first Snapshot measures CPU for, as there is
// no earlier snapshot to compare with.
const firstSample = 250 * time.Millisecond

// cpuTimes is the CPU a process has used and when it started, both in
// clock ticks. The start time tells a reused PID apart.
type cpuTimes struct{ used, start uint64 }

var cpuState struct {
	sync.Mutex
	prev map[int]cpuTimes
	at   time.Time
}

// recentCPU replaces each process's CPU with its usage since the previous
// Snapshot. Linux ps averages CPU over the whole life of a process, so an
// agent that sat idle for an hour and is busy now would still look idle;
// macOS ps already reports recent usage.
func recentCPU(procs []Proc) {
	cur, ok := readCPUTimes(procs)
	if !ok {
		return
	}
	now := time.Now()
	cpuState.Lock()
	defer cpuState.Unlock()
	if cpuState.prev == nil {
		time.Sleep(firstSample)
		cpuState.prev, cpuState.at = cur, now
		cur, _ = readCPUTimes(procs)
		now = time.Now()
	}
	applyCPU(procs, cpuState.prev, cur, now.Sub(cpuState.at))
	cpuState.prev, cpuState.at = cur, now
}

// applyCPU sets CPU from the ticks used between two samples taken elapsed
// apart. A process missing from either sample, or started again under the
// same PID, keeps the value ps gave it.
func applyCPU(procs []Proc, prev, cur map[int]cpuTimes, elapsed time.Duration) {
	if elapsed <= 0 {
		return
	}
	for i := range procs {
		a, okA := prev[procs[i].PID]
		b, okB := cur[procs[i].PID]
		if !okA || !okB || a.start != b.start || b.used < a.used {
			continue
		}
		procs[i].CPU = float64(b.used-a.used) / clockTicks / elapsed.Seconds() * 100
	}
}

// readCPUTimes reads /proc/<pid>/stat for each process. It returns false
// when there is no /proc.
func readCPUTimes(procs []Proc) (map[int]cpuTimes, bool) {
	if _, err := os.Stat(procRoot + "/self/stat"); err != nil {
		return nil, false
	}
	times := make(map[int]cpuTimes, len(procs))
	for _, p := range procs {
		data, err := os.ReadFile(procRoot + "/" + strconv.Itoa(p.PID) + "/stat")
		if err != nil {
			continue // exited since ps ran
		}
		if t, ok := parseStat(data); ok {
			times[p.PID] = t
		}
	}
	return times, true
}

// parseStat reads utime, stime and starttime from a /proc/<pid>/stat line.
// The command name in parentheses may hold spaces and parentheses, so the
// fields are counted from the last ')'.
func parseStat(data []byte) (cpuTimes, bool) {
	s := string(data)
	i := strings.LastIndexByte(s, ')')
	if i < 0 {
		return cpuTimes{}, false
	}
	f := strings.Fields(s[i+1:]) // f[0] is field 3, state
	if len(f) < 20 {
		return cpuTimes{}, false
	}
	utime, err1 := strconv.ParseUint(f[11], 10, 64)
	stime, err2 := strconv.ParseUint(f[12], 10, 64)
	start, err3 := strconv.ParseUint(f[19], 10, 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return cpuTimes{}, false
	}
	return cpuTimes{used: utime + stime, start: start}, true
}

// selfPIDLevels is how deep lazyai's own PID namespace is nested.
var selfPIDLevels = sync.OnceValue(func() int { return pidLevels("self") })

// inContainer reports whether pid runs in a PID namespace nested below
// lazyai's own, as the processes inside a docker or podman container do
// on Linux. The docker client that started the container already stands
// for it. Without /proc it returns false.
func inContainer(pid int) bool {
	self := selfPIDLevels()
	return self > 0 && pidLevels(strconv.Itoa(pid)) > self
}

// pidLevels returns the number of PID namespaces pid is visible in, or 0
// when it is unknown. /proc/<pid>/status is readable for every process.
func pidLevels(pid string) int {
	data, err := os.ReadFile(procRoot + "/" + pid + "/status")
	if err != nil {
		return 0
	}
	return parseNSpid(data)
}

// parseNSpid counts the PIDs on the NSpid line of /proc/<pid>/status: one
// per PID namespace, from the outermost in.
func parseNSpid(status []byte) int {
	sc := bufio.NewScanner(bytes.NewReader(status))
	for sc.Scan() {
		if rest, ok := strings.CutPrefix(sc.Text(), "NSpid:"); ok {
			return len(strings.Fields(rest))
		}
	}
	return 0
}
