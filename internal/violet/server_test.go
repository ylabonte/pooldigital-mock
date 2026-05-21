package violet

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestServer(t *testing.T) (*httptest.Server, *State) {
	t.Helper()
	seed, err := LoadSeed()
	if err != nil {
		t.Fatalf("LoadSeed: %v", err)
	}
	fixed := time.Unix(1_700_000_000, 0).UTC()
	st := New(Config{Seed: seed, Clock: func() time.Time { return fixed }})
	srv := httptest.NewServer(NewHandler(ServerConfig{State: st, Username: "admin", Password: "admin"}))
	t.Cleanup(srv.Close)
	return srv, st
}

func TestGetReadingsNoAuthRequired(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/getReadings?ALL")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := body["pH_value_min"]; !ok {
		t.Error("missing pH_value_min in /getReadings?ALL response")
	}
}

func TestGetReadingsMissingQuery500(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/getReadings")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", resp.StatusCode)
	}
}

func TestGetReadingsSelective(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/getReadings?pH_value")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Should include pH_value_min and pH_value_max from the seed.
	if _, ok := body["pH_value_min"]; !ok {
		t.Error("missing pH_value_min in selective response")
	}
	// Should NOT include unrelated keys
	if _, ok := body["orp_value_min"]; ok {
		t.Error("selective response leaked orp_value_min")
	}
}

func TestGetConfigRequiresAuth(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/getConfig")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestGetConfigWithAuth(t *testing.T) {
	srv, _ := newTestServer(t)
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/getConfig", nil)
	req.SetBasicAuth("admin", "admin")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestSetFunctionManually(t *testing.T) {
	srv, st := newTestServer(t)
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/setFunctionManually?PUMP,ON,0,0", nil)
	req.SetBasicAuth("admin", "admin")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if st.Snapshot()["PUMP"] != int(OutputManualOn) {
		t.Errorf("PUMP not switched on: %v", st.Snapshot()["PUMP"])
	}
}

func TestSetFunctionUnknownKey(t *testing.T) {
	srv, _ := newTestServer(t)
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/setFunctionManually?XYZ_FAKE,ON", nil)
	req.SetBasicAuth("admin", "admin")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestSetFunctionBadDuration(t *testing.T) {
	srv, _ := newTestServer(t)
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/setFunctionManually?PUMP,ON,abc,0", nil)
	req.SetBasicAuth("admin", "admin")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestSetTargetValues(t *testing.T) {
	srv, st := newTestServer(t)
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/setTargetValues?target=pH&value=7.2", nil)
	req.SetBasicAuth("admin", "admin")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if st.Snapshot()["pH_target"] != 7.2 {
		t.Errorf("pH_target = %v, want 7.2", st.Snapshot()["pH_target"])
	}
}

func TestSetTargetMissingParams(t *testing.T) {
	srv, _ := newTestServer(t)
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/setTargetValues?target=pH", nil)
	req.SetBasicAuth("admin", "admin")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestSetConfigPersists(t *testing.T) {
	srv, st := newTestServer(t)
	body := strings.NewReader(`{"foo": "bar", "n": 7}`)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/setConfig", body)
	req.SetBasicAuth("admin", "admin")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	cfg := st.ConfigSnapshot()
	if cfg["foo"] != "bar" {
		t.Errorf("config[foo] = %v, want bar", cfg["foo"])
	}
}

func TestSetConfigBadJSON(t *testing.T) {
	srv, _ := newTestServer(t)
	body := strings.NewReader(`not json`)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/setConfig", body)
	req.SetBasicAuth("admin", "admin")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestSetConfigArrayBody(t *testing.T) {
	srv, _ := newTestServer(t)
	body := strings.NewReader(`[1, 2, 3]`)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/setConfig", body)
	req.SetBasicAuth("admin", "admin")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestSetDosingParameters(t *testing.T) {
	srv, st := newTestServer(t)
	body := strings.NewReader(`{"DOS_CL_max": 3.0}`)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/setDosingParameters", body)
	req.SetBasicAuth("admin", "admin")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if got := st.DosingParametersSnapshot()["DOS_CL_max"]; got == nil {
		t.Error("DOS_CL_max not persisted")
	}
}

func TestMethodNotAllowed(t *testing.T) {
	srv, _ := newTestServer(t)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/getReadings?ALL", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", resp.StatusCode)
	}
}

func TestBadCredentials(t *testing.T) {
	srv, _ := newTestServer(t)
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/getConfig", nil)
	req.SetBasicAuth("nope", "nope")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = body
}
