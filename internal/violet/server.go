package violet

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// Endpoint paths — copied from myviolet.constants.
const (
	pathGetReadings     = "/getReadings"
	pathGetConfig       = "/getConfig"
	pathSetFunction     = "/setFunctionManually"
	pathSetTargetValues = "/setTargetValues"
	pathSetConfig       = "/setConfig"
	pathSetDosingParams = "/setDosingParameters"
)

// ServerConfig configures a new violet HTTP server.
type ServerConfig struct {
	State    *State
	Username string
	Password string
}

// NewHandler builds the http.Handler for the violet mock.
func NewHandler(cfg ServerConfig) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(pathGetReadings, methodOnly(http.MethodGet, handleGetReadings(cfg.State)))
	mux.HandleFunc(pathGetConfig, methodOnly(http.MethodGet, handleGetConfig(cfg.State)))
	mux.HandleFunc(pathSetFunction, methodOnly(http.MethodGet, handleSetFunction(cfg.State)))
	mux.HandleFunc(pathSetTargetValues, methodOnly(http.MethodGet, handleSetTarget(cfg.State)))
	mux.HandleFunc(pathSetConfig, methodOnly(http.MethodPost, handleSetConfig(cfg.State)))
	mux.HandleFunc(pathSetDosingParams, methodOnly(http.MethodPost, handleSetDosingParameters(cfg.State)))
	return basicAuth(cfg.Username, cfg.Password, mux)
}

func methodOnly(method string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		next(w, r)
	}
}

// basicAuth gates every path except /getReadings.
func basicAuth(user, pass string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == pathGetReadings {
			next.ServeHTTP(w, r)
			return
		}
		u, p, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(u), []byte(user)) != 1 ||
			subtle.ConstantTimeCompare([]byte(p), []byte(pass)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="violet"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}

func handleGetReadings(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.RawQuery
		if query == "" {
			http.Error(w, "missing query parameters", http.StatusInternalServerError)
			return
		}
		snap := state.Snapshot()
		snap = ApplyDrift(snap, state.ElapsedSeconds())

		keywords := make(map[string]struct{})
		for _, part := range strings.Split(query, ",") {
			if part != "" {
				keywords[part] = struct{}{}
			}
		}
		if _, ok := keywords["ALL"]; ok {
			writeJSON(w, snap)
			return
		}

		selected := make(map[string]any)
		for key := range keywords {
			for _, full := range []string{key, key + "_min", key + "_max"} {
				if v, ok := snap[full]; ok {
					selected[full] = v
				}
			}
		}
		writeJSON(w, selected)
	}
}

func handleGetConfig(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, state.ConfigSnapshot())
	}
}

func handleSetFunction(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := r.URL.RawQuery
		parts := strings.Split(raw, ",")
		if len(parts) < 2 {
			http.Error(w, "expected KEY,ACTION,DURATION,VALUE", http.StatusBadRequest)
			return
		}
		key := parts[0]
		action := parts[1]
		duration, value := 0, 0
		if len(parts) > 2 && parts[2] != "" {
			n, err := strconv.Atoi(parts[2])
			if err != nil {
				http.Error(w, "duration and value must be integers", http.StatusBadRequest)
				return
			}
			duration = n
		}
		if len(parts) > 3 && parts[3] != "" {
			n, err := strconv.Atoi(parts[3])
			if err != nil {
				http.Error(w, "duration and value must be integers", http.StatusBadRequest)
				return
			}
			value = n
		}
		if err := state.ApplySetFunction(key, action, duration, value); err != nil {
			if errors.Is(err, ErrUnknownControlKey) {
				http.Error(w, "unknown control key", http.StatusBadRequest)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	}
}

func handleSetTarget(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target := r.URL.Query().Get("target")
		valueRaw := r.URL.Query().Get("value")
		if target == "" || valueRaw == "" {
			http.Error(w, "missing target or value", http.StatusBadRequest)
			return
		}
		value, err := strconv.ParseFloat(valueRaw, 64)
		if err != nil {
			http.Error(w, "value must be numeric", http.StatusBadRequest)
			return
		}
		state.ApplySetTarget(target, value)
		writeJSON(w, map[string]any{"ok": true})
	}
}

func handleSetConfig(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := readJSONObject(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		state.ApplySetConfig(payload)
		writeJSON(w, map[string]any{"ok": true})
	}
}

func handleSetDosingParameters(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := readJSONObject(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		state.ApplySetDosingParameters(payload)
		writeJSON(w, map[string]any{"ok": true})
	}
}

func readJSONObject(r *http.Request) (map[string]any, error) {
	defer func() { _ = r.Body.Close() }()
	dec := json.NewDecoder(r.Body)
	dec.UseNumber()
	var payload any
	if err := dec.Decode(&payload); err != nil {
		return nil, errors.New("invalid JSON body")
	}
	m, ok := payload.(map[string]any)
	if !ok {
		return nil, errors.New("JSON body must be an object")
	}
	return m, nil
}
