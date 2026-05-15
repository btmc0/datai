//go:build darwin

package hostmetrics

import "testing"

func TestParseVMStat(t *testing.T) {
	pageSize, freePages, err := parseVMStat(`Mach Virtual Memory Statistics: (page size of 16384 bytes)
Pages free:                               100.
Pages active:                             200.
Pages speculative:                         25.
`)
	if err != nil {
		t.Fatal(err)
	}
	if pageSize != 16384 {
		t.Fatalf("pageSize = %d, want 16384", pageSize)
	}
	if freePages != 125 {
		t.Fatalf("freePages = %d, want 125", freePages)
	}
}
