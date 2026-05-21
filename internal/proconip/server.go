package proconip

import (
	"crypto/subtle"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ServerConfig configures a new mock server.
type ServerConfig struct {
	State    *State
	Username string
	Password string
}

// NewHandler builds an http.Handler exposing the ProCon.IP routes.
func NewHandler(cfg ServerConfig) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/GetState.csv", methodOnly(http.MethodGet, handleGetState(cfg.State)))
	mux.HandleFunc("/GetDmx.csv", methodOnly(http.MethodGet, handleGetDMX(cfg.State)))
	mux.HandleFunc("/usrcfg.cgi", methodOnly(http.MethodPost, handleUsrcfg(cfg.State)))
	mux.HandleFunc("/Command.htm", methodOnly(http.MethodGet, handleCommand))
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

func basicAuth(user, pass string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(u), []byte(user)) != 1 ||
			subtle.ConstantTimeCompare([]byte(p), []byte(pass)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="ProCon.IP"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleGetState(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		body, err := RenderGetState(state, time.Time{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		_, _ = w.Write([]byte(body))
	}
}

func handleGetDMX(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		_, _ = w.Write([]byte(RenderGetDMX(state)))
	}
}

func handleUsrcfg(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form body", http.StatusBadRequest)
			return
		}
		if ena := r.PostForm.Get("ENA"); ena != "" {
			enableStr, onStr, ok := strings.Cut(ena, ",")
			if !ok {
				http.Error(w, "Invalid ENA payload", http.StatusBadRequest)
				return
			}
			enableMask, err1 := strconv.Atoi(strings.TrimSpace(enableStr))
			onMask, err2 := strconv.Atoi(strings.TrimSpace(onStr))
			if err1 != nil || err2 != nil {
				http.Error(w, "Invalid ENA payload", http.StatusBadRequest)
				return
			}
			if err := state.ApplyENA(enableMask, onMask); err != nil {
				http.Error(w, "Invalid ENA payload", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte("OK"))
			return
		}
		low := r.PostForm.Get("CH1_8")
		high := r.PostForm.Get("CH9_16")
		if low != "" && high != "" {
			lowVals, err1 := parseIntList(low)
			highVals, err2 := parseIntList(high)
			if err1 != nil || err2 != nil {
				http.Error(w, "Invalid DMX payload", http.StatusBadRequest)
				return
			}
			if err := state.ApplyDMX(lowVals, highVals); err != nil {
				http.Error(w, "Invalid DMX payload", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte("OK"))
			return
		}
		http.Error(w, "Unrecognized usrcfg.cgi payload", http.StatusBadRequest)
	}
}

func handleCommand(w http.ResponseWriter, r *http.Request) {
	dosage := r.URL.Query().Get("MAN_DOSAGE")
	if dosage == "" {
		http.Error(w, "Missing MAN_DOSAGE query parameter", http.StatusBadRequest)
		return
	}
	target, duration, ok := strings.Cut(dosage, ",")
	if !ok {
		http.Error(w, "Invalid MAN_DOSAGE payload", http.StatusBadRequest)
		return
	}
	if _, err := strconv.Atoi(strings.TrimSpace(target)); err != nil {
		http.Error(w, "Invalid MAN_DOSAGE payload", http.StatusBadRequest)
		return
	}
	if _, err := strconv.Atoi(strings.TrimSpace(duration)); err != nil {
		http.Error(w, "Invalid MAN_DOSAGE payload", http.StatusBadRequest)
		return
	}
	_, _ = w.Write([]byte("OK"))
}

func parseIntList(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	out := make([]int, len(parts))
	for i, p := range parts {
		v, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}
