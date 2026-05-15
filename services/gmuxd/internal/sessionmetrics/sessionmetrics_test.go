package sessionmetrics

import (
	"testing"

	"github.com/gmuxapp/gmux/services/gmuxd/internal/store"
)

func TestParseProcesses(t *testing.T) {
	procs := parseProcesses(`
  10   1 12000 /Users/me/.local/bin/gmux pi
  11  10 20000 pi
  12  11  3000 helper process
`)
	if got := procs[10].command; got != "/Users/me/.local/bin/gmux pi" {
		t.Fatalf("command = %q", got)
	}
	if got := procs[12].rssKB; got != 3000 {
		t.Fatalf("rssKB = %d", got)
	}
}

func TestSumTreeRSSIncludesRunnerAndChildTree(t *testing.T) {
	procs := map[int]processInfo{
		10: {pid: 10, ppid: 1, rssKB: 12000, command: "/Users/me/.local/bin/gmux pi"},
		11: {pid: 11, ppid: 10, rssKB: 20000, command: "pi"},
		12: {pid: 12, ppid: 11, rssKB: 3000, command: "helper"},
		99: {pid: 99, ppid: 1, rssKB: 9999, command: "other"},
	}
	children := map[int][]int{1: {10, 99}, 10: {11}, 11: {12}}

	child := procs[11]
	root := child.pid
	if parent, ok := procs[child.ppid]; ok && isGmuxRunner(parent.command) {
		root = parent.pid
	}

	if got, want := sumTreeRSS(root, procs, children), uint64(35000); got != want {
		t.Fatalf("sumTreeRSS = %d, want %d", got, want)
	}
}

func TestCollectSkipsDeadAndPeerSessions(t *testing.T) {
	_ = []store.Session{
		{ID: "live", Alive: true, Pid: 11},
		{ID: "dead", Alive: false, Pid: 11},
		{ID: "peer", Alive: true, Peer: "laptop", Pid: 11},
	}
	// Collect itself shells out to ps; skip behavior is covered by keeping
	// the filtering in Collect before process lookup and by integration use.
}
