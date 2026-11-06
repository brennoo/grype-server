package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockScanner struct {
	result          []byte
	err             error
	gotVexDocuments [][]byte
}

func (m *mockScanner) Scan(_ []byte, vexDocuments [][]byte) ([]byte, error) {
	m.gotVexDocuments = vexDocuments
	return m.result, m.err
}

func newTestServer(t *testing.T, sc *mockScanner) *httptest.Server {
	t.Helper()
	return httptest.NewServer(newAPI(sc))
}

func TestHandler_Success(t *testing.T) {
	expected := []byte(`{"matches":[]}`)
	ts := newTestServer(t, &mockScanner{result: expected})
	defer ts.Close()

	resp := doScan(t, ts.URL, `{"sbom":"e30="}`) // e30= is base64({})
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body struct {
		Vulnerabilities []byte `json:"vulnerabilities"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if string(body.Vulnerabilities) != string(expected) {
		t.Errorf("got %s, want %s", body.Vulnerabilities, expected)
	}
}

func TestHandler_VexDocumentsPassthrough(t *testing.T) {
	sc := &mockScanner{result: []byte(`{"matches":[]}`)}
	ts := newTestServer(t, sc)
	defer ts.Close()

	// "e30=" is base64({})
	resp := doScan(t, ts.URL, `{"sbom":"e30=","vexDocuments":["e30="]}`)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(sc.gotVexDocuments) != 1 || string(sc.gotVexDocuments[0]) != "{}" {
		t.Errorf("expected vexDocuments to be passed through to Scan, got %v", sc.gotVexDocuments)
	}
}

func TestHandler_ScanError(t *testing.T) {
	ts := newTestServer(t, &mockScanner{err: errors.New("scan failed")})
	defer ts.Close()

	resp := doScan(t, ts.URL, `{"sbom":"e30="}`)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestHandler_InvalidJSON(t *testing.T) {
	ts := newTestServer(t, &mockScanner{})
	defer ts.Close()

	resp := doScan(t, ts.URL, `not-json`)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func doScan(t *testing.T, baseURL, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/scanSBOM", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}
