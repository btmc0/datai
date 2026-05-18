package main

// runAfterDiscoveryFirstScan starts dependent maintenance only after discovery's
// initial scan completes. sessionmeta restores persisted sessions as dead, so
// retention cleanup must not prune them before discovery can resurrect any
// still-live runner sockets.
func runAfterDiscoveryFirstScan(discoveryFirstScan <-chan struct{}, stop <-chan struct{}, run func()) {
	select {
	case <-discoveryFirstScan:
		run()
	case <-stop:
	}
}
