package proconip

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) (*httptest.Server, *State) {
	t.Helper()
	st := New(Config{})
	srv := httptest.NewServer(NewHandler(ServerConfig{State: st, Username: "admin", Password: "admin"}))
	t.Cleanup(srv.Close)
	return srv, st
}

func doRequest(t *testing.T, srv *httptest.Server, method, path string, body io.Reader, withAuth bool) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, srv.URL+path, body)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if withAuth {
		req.SetBasicAuth("admin", "admin")
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	return resp
}

func TestGetStateRequiresAuth(t *testing.T) {
	srv, _ := newTestServer(t)
	resp := doRequest(t, srv, http.MethodGet, "/GetState.csv", nil, false)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
	if got := resp.Header.Get("WWW-Authenticate"); !strings.Contains(got, "Basic") {
		t.Errorf("missing or bad WWW-Authenticate: %q", got)
	}
}

func TestGetStateReturnsCSV(t *testing.T) {
	srv, _ := newTestServer(t)
	resp := doRequest(t, srv, http.MethodGet, "/GetState.csv", nil, true)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "text/csv") {
		t.Errorf("Content-Type = %q, want text/csv", resp.Header.Get("Content-Type"))
	}
	b, _ := io.ReadAll(resp.Body)
	rows := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	if len(rows) != 6 {
		t.Errorf("expected 6 rows, got %d", len(rows))
	}
	if !strings.HasPrefix(rows[0], "SYSINFO,") {
		t.Errorf("row 0 = %q", rows[0])
	}
}

func TestGetDMXReturnsCSV(t *testing.T) {
	srv, _ := newTestServer(t)
	resp := doRequest(t, srv, http.MethodGet, "/GetDmx.csv", nil, true)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	parts := strings.Split(strings.TrimRight(string(b), "\n"), ",")
	if len(parts) != NumDMXChannels {
		t.Errorf("expected %d DMX channels, got %d", NumDMXChannels, len(parts))
	}
}

func TestUsrcfgENAUpdatesRelay(t *testing.T) {
	srv, st := newTestServer(t)
	form := url.Values{"ENA": {"1,1"}, "MANUAL": {"1"}}
	resp := doRequest(t, srv, http.MethodPost, "/usrcfg.cgi", strings.NewReader(form.Encode()), true)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := st.CSVRelayValue(0); got != 3 {
		t.Errorf("relay bit 0 = %d, want 3", got)
	}
}

func TestUsrcfgDMXUpdatesChannels(t *testing.T) {
	srv, st := newTestServer(t)
	form := url.Values{
		"CH1_8":  {"255,0,0,0,0,0,0,0"},
		"CH9_16": {"0,0,0,0,0,0,0,0"},
	}
	resp := doRequest(t, srv, http.MethodPost, "/usrcfg.cgi", strings.NewReader(form.Encode()), true)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := st.CopyDMX(); got[0] != 255 {
		t.Errorf("dmx[0] = %d, want 255", got[0])
	}
}

func TestUsrcfgBadENA(t *testing.T) {
	srv, _ := newTestServer(t)
	form := url.Values{"ENA": {"notanumber,1"}}
	resp := doRequest(t, srv, http.MethodPost, "/usrcfg.cgi", strings.NewReader(form.Encode()), true)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestUsrcfgUnrecognizedPayload(t *testing.T) {
	srv, _ := newTestServer(t)
	resp := doRequest(t, srv, http.MethodPost, "/usrcfg.cgi", strings.NewReader("FOO=bar"), true)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestCommandManualDosage(t *testing.T) {
	srv, _ := newTestServer(t)
	resp := doRequest(t, srv, http.MethodGet, "/Command.htm?MAN_DOSAGE=0,60", nil, true)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestCommandMissingDosage(t *testing.T) {
	srv, _ := newTestServer(t)
	resp := doRequest(t, srv, http.MethodGet, "/Command.htm", nil, true)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestCommandBadDosage(t *testing.T) {
	srv, _ := newTestServer(t)
	cases := []string{"", "0", "0,abc", "abc,60"}
	for _, dosage := range cases {
		path := "/Command.htm?MAN_DOSAGE=" + url.QueryEscape(dosage)
		resp := doRequest(t, srv, http.MethodGet, path, nil, true)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("dosage=%q: status = %d, want 400", dosage, resp.StatusCode)
		}
	}
}

func TestMethodNotAllowed(t *testing.T) {
	srv, _ := newTestServer(t)
	resp := doRequest(t, srv, http.MethodPut, "/GetState.csv", nil, true)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", resp.StatusCode)
	}
}

func TestBadCredentials(t *testing.T) {
	srv, _ := newTestServer(t)
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/GetState.csv", nil)
	req.SetBasicAuth("evil", "hacker")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}
