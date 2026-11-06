//go:build integration

package scanner

import (
	"os"
	"testing"
)

// Run with: go test -tags integration ./pkg/scanner/... -v
// Requires GRYPE_DB_DIR env var pointing to a directory with a downloaded Grype DB.
func TestScanSbomJson_Integration(t *testing.T) {
	dbDir := os.Getenv("GRYPE_DB_DIR")
	if dbDir == "" {
		t.Skip("GRYPE_DB_DIR not set")
	}

	s, err := Create(&Config{DbRootDir: dbDir})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ctx := t.Context()

	errChan := make(chan struct{}, 1)
	if err := s.Start(ctx, errChan); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()

	// log4j 2.14.1 is known vulnerable (Log4Shell CVE-2021-44228)
	sbom := `{"bomFormat":"CycloneDX","specVersion":"1.4","version":1,"components":[{"type":"library","name":"log4j-core","version":"2.14.1","purl":"pkg:maven/org.apache.logging.log4j/log4j-core@2.14.1"}]}`

	doc, err := s.ScanSbomJson(sbom, nil)
	if err != nil {
		t.Fatalf("ScanSbomJson: %v", err)
	}
	if len(doc.Matches) == 0 {
		t.Error("expected vulnerabilities for log4j 2.14.1, got none")
	}

	vexDoc, err := os.ReadFile("testdata/vex-not-affected.json")
	if err != nil {
		t.Fatalf("failed to read vex fixture: %v", err)
	}

	docWithVex, err := s.ScanSbomJson(sbom, [][]byte{vexDoc})
	if err != nil {
		t.Fatalf("ScanSbomJson with vex: %v", err)
	}
	for _, m := range docWithVex.Matches {
		if m.Vulnerability.ID == "CVE-2021-44228" {
			t.Errorf("expected CVE-2021-44228 to be suppressed by vex, but it was still present in matches")
		}
	}
}
