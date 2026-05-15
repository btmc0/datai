package sessionmetrics

import (
	"context"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gmuxapp/gmux/services/gmuxd/internal/store"
)

type Metric struct {
	RSSBytes uint64 `json:"rss_bytes"`
}

type processInfo struct {
	pid     int
	ppid    int
	rssKB   uint64
	command string
}

func Collect(ctx context.Context, sessions []store.Session) map[string]Metric {
	procs, err := readProcesses(ctx)
	if err != nil {
		return map[string]Metric{}
	}

	children := make(map[int][]int)
	for pid, p := range procs {
		children[p.ppid] = append(children[p.ppid], pid)
	}

	out := make(map[string]Metric)
	for _, s := range sessions {
		if !s.Alive || s.Peer != "" || s.Pid <= 0 {
			continue
		}
		child, ok := procs[s.Pid]
		if !ok {
			continue
		}

		root := child.pid
		if parent, ok := procs[child.ppid]; ok && isGmuxRunner(parent.command) {
			root = parent.pid
		}

		rssKB := sumTreeRSS(root, procs, children)
		if rssKB > 0 {
			out[s.ID] = Metric{RSSBytes: rssKB * 1024}
		}
	}
	return out
}

func readProcesses(ctx context.Context) (map[int]processInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "ps", "-axo", "pid=,ppid=,rss=,command=").Output()
	if err != nil {
		return nil, err
	}
	return parseProcesses(string(out)), nil
}

func parseProcesses(out string) map[int]processInfo {
	procs := make(map[int]processInfo)
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		pid, err1 := strconv.Atoi(fields[0])
		ppid, err2 := strconv.Atoi(fields[1])
		rssKB, err3 := strconv.ParseUint(fields[2], 10, 64)
		if err1 != nil || err2 != nil || err3 != nil || pid <= 0 {
			continue
		}
		procs[pid] = processInfo{
			pid:     pid,
			ppid:    ppid,
			rssKB:   rssKB,
			command: strings.Join(fields[3:], " "),
		}
	}
	return procs
}

func isGmuxRunner(command string) bool {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return false
	}
	return filepath.Base(fields[0]) == "gmux"
}

func sumTreeRSS(root int, procs map[int]processInfo, children map[int][]int) uint64 {
	var total uint64
	seen := map[int]bool{}
	stack := []int{root}
	for len(stack) > 0 {
		pid := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[pid] {
			continue
		}
		seen[pid] = true
		p, ok := procs[pid]
		if !ok {
			continue
		}
		total += p.rssKB
		stack = append(stack, children[pid]...)
	}
	return total
}
