package agent

import (
	"bufio"
	"bytes"
	"os/exec"
	"strconv"
	"strings"
)

// workingDirs asks lsof for the current directory of each PID in one call.
// PIDs lsof cannot see are missing from the result.
func workingDirs(pids []int) map[int]string {
	dirs := map[int]string{}
	if len(pids) == 0 {
		return dirs
	}
	list := make([]string, len(pids))
	for i, pid := range pids {
		list[i] = strconv.Itoa(pid)
	}
	// lsof exits 1 when some PIDs are gone; the output is still usable.
	out, _ := exec.Command("lsof", "-a", "-d", "cwd", "-Fn", "-p", strings.Join(list, ",")).Output()
	return parseLsof(out)
}

func parseLsof(out []byte) map[int]string {
	dirs := map[int]string{}
	pid := 0
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		switch line[0] {
		case 'p':
			pid, _ = strconv.Atoi(line[1:])
		case 'n':
			if pid != 0 {
				dirs[pid] = line[1:]
			}
		}
	}
	return dirs
}
