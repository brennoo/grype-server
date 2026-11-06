package scanner

import (
	"testing"
)

func TestScanSbomJson_NilProvider(t *testing.T) {
	s := &Scanner{}
	_, err := s.ScanSbomJson("any-sbom", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	const want = "vulnerability provider wasn't set"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}
