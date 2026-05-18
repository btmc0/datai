package main

import (
	"testing"
	"time"
)

func TestRunAfterDiscoveryFirstScanWaitsForDiscovery(t *testing.T) {
	discoveryFirstScan := make(chan struct{})
	stop := make(chan struct{})
	ran := make(chan struct{})

	go runAfterDiscoveryFirstScan(discoveryFirstScan, stop, func() {
		close(ran)
	})

	select {
	case <-ran:
		t.Fatal("scanner ran before discovery first scan completed")
	default:
	}

	close(discoveryFirstScan)
	select {
	case <-ran:
	case <-time.After(time.Second):
		t.Fatal("scanner did not run after discovery first scan completed")
	}
}

func TestRunAfterDiscoveryFirstScanStopsBeforeDiscovery(t *testing.T) {
	discoveryFirstScan := make(chan struct{})
	stop := make(chan struct{})
	ran := make(chan struct{})
	done := make(chan struct{})

	go func() {
		runAfterDiscoveryFirstScan(discoveryFirstScan, stop, func() {
			close(ran)
		})
		close(done)
	}()

	close(stop)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scanner gate did not stop")
	}

	select {
	case <-ran:
		t.Fatal("scanner ran after stop before discovery first scan")
	default:
	}
}
