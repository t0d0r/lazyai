package agent

import (
	"bufio"
	"bytes"
	"maps"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// workingDirs returns the current directory of each PID. On Linux it reads
// the /proc/<pid>/cwd links; for PIDs that leaves out (and on macOS, which
// has no /proc) it asks lsof in one call, if lsof is installed. PIDs
// neither can see are missing from the result.
func workingDirs(pids []int) map[int]string {
	dirs := map[int]string{}
	var list []string
	for _, pid := range pids {
		if dir, err := os.Readlink(procRoot + "/" + strconv.Itoa(pid) + "/cwd"); err == nil {
			dirs[pid] = dir
		} else {
			list = append(list, strconv.Itoa(pid))
		}
	}
	if len(list) == 0 {
		return dirs
	}
	if _, err := exec.LookPath("lsof"); err != nil {
		return dirs
	}
	// lsof exits 1 when some PIDs are gone; the output is still usable.
	out, _ := exec.Command("lsof", "-a", "-d", "cwd", "-Fn", "-p", strings.Join(list, ",")).Output()
	maps.Copy(dirs, parseLsof(out))
	return dirs
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
