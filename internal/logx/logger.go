// Package logx is the shared structured logger and HTTP middleware used by
// both mock servers. It produces a single colorful stream of per-request
// lines plus startup banners.
package logx

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Logger renders banners and per-request log lines to an io.Writer.
// Concurrent writes from request middleware are serialized through mu so
// lines never interleave.
type Logger struct {
	mu     sync.Mutex
	out    io.Writer
	quiet  bool
	styles styles
}

// styles holds the lipgloss renderers used for output. They are computed
// once at logger construction so per-request rendering is allocation-light.
type styles struct {
	banner   lipgloss.Style
	bannerHL lipgloss.Style
	proconip lipgloss.Style
	violet   lipgloss.Style
	method   lipgloss.Style
	path     lipgloss.Style
	src      lipgloss.Style
	ts       lipgloss.Style
	arrow    lipgloss.Style
	dim      lipgloss.Style
	statusOK lipgloss.Style
	status3  lipgloss.Style
	status4  lipgloss.Style
	status5  lipgloss.Style
}

// Options configures a Logger.
type Options struct {
	// Out is where banners and request lines are written. Defaults to
	// os.Stdout when nil.
	Out io.Writer
	// NoColor disables ANSI styling.
	NoColor bool
	// Quiet suppresses per-request lines; banners still render.
	Quiet bool
}

// New constructs a Logger with the given options.
func New(opts Options) *Logger {
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}
	profile := lipgloss.ColorProfile()
	if opts.NoColor || os.Getenv("NO_COLOR") != "" {
		profile = 0 // Ascii — strips ANSI
	}
	r := lipgloss.NewRenderer(out)
	r.SetColorProfile(profile)

	s := styles{
		banner:   r.NewStyle().Foreground(lipgloss.Color("99")).Bold(true),
		bannerHL: r.NewStyle().Foreground(lipgloss.Color("212")).Bold(true),
		proconip: r.NewStyle().Foreground(lipgloss.Color("39")).Bold(true),  // cyan
		violet:   r.NewStyle().Foreground(lipgloss.Color("165")).Bold(true), // magenta
		method:   r.NewStyle().Foreground(lipgloss.Color("250")).Bold(true),
		path:     r.NewStyle().Foreground(lipgloss.Color("252")),
		src:      r.NewStyle().Foreground(lipgloss.Color("245")),
		ts:       r.NewStyle().Foreground(lipgloss.Color("244")),
		arrow:    r.NewStyle().Foreground(lipgloss.Color("241")),
		dim:      r.NewStyle().Foreground(lipgloss.Color("241")),
		statusOK: r.NewStyle().Foreground(lipgloss.Color("42")).Bold(true),  // green
		status3:  r.NewStyle().Foreground(lipgloss.Color("33")).Bold(true),  // blue
		status4:  r.NewStyle().Foreground(lipgloss.Color("214")).Bold(true), // yellow
		status5:  r.NewStyle().Foreground(lipgloss.Color("196")).Bold(true), // red
	}
	return &Logger{out: out, quiet: opts.Quiet, styles: s}
}

// BannerLine is one row of the startup banner — a name and the URL it
// serves on.
type BannerLine struct {
	Name string
	URL  string
}

// Banner renders the startup banner. Lines are shown in the order given.
func (l *Logger) Banner(title string, lines []BannerLine) {
	l.mu.Lock()
	defer l.mu.Unlock()

	width := len(title) + 4
	for _, ln := range lines {
		w := len(ln.Name) + len(ln.URL) + 6
		if w > width {
			width = w
		}
	}
	inner := width - 2

	top := "┌" + strings.Repeat("─", inner) + "┐"
	bot := "└" + strings.Repeat("─", inner) + "┘"
	titleLine := " " + title + " "
	titleLine = "│" + titleLine + strings.Repeat(" ", inner-len(titleLine)) + "│"

	_, _ = fmt.Fprintln(l.out, l.styles.banner.Render(top))
	_, _ = fmt.Fprintln(l.out, l.styles.bannerHL.Render(titleLine))
	sep := "├" + strings.Repeat("─", inner) + "┤"
	_, _ = fmt.Fprintln(l.out, l.styles.banner.Render(sep))
	for _, ln := range lines {
		text := fmt.Sprintf(" %-9s %s", ln.Name, ln.URL)
		pad := inner - len(text)
		if pad < 0 {
			pad = 0
		}
		body := "│" + text + strings.Repeat(" ", pad) + "│"
		_, _ = fmt.Fprintln(l.out, l.styles.banner.Render(body))
	}
	_, _ = fmt.Fprintln(l.out, l.styles.banner.Render(bot))
}

// Info prints an informational line outside the request flow (e.g. shutdown
// notices). It respects the quiet flag.
func (l *Logger) Info(format string, args ...any) {
	if l.quiet {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = fmt.Fprintln(l.out, l.styles.dim.Render(fmt.Sprintf(format, args...)))
}

// LogRequest writes a single colored request line. It is called by the
// middleware after the handler returns.
func (l *Logger) LogRequest(mock string, srcIP string, method, path string, status int, dur time.Duration) {
	if l.quiet {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	mockStyle := l.styles.proconip
	if mock == "violet" {
		mockStyle = l.styles.violet
	}

	statusStyle := l.styles.statusOK
	switch {
	case status >= 500:
		statusStyle = l.styles.status5
	case status >= 400:
		statusStyle = l.styles.status4
	case status >= 300:
		statusStyle = l.styles.status3
	}

	ts := time.Now().Format("15:04:05.000")
	_, _ = fmt.Fprintf(l.out,
		"%s  %s  %s %s %s %s  %s  %s\n",
		l.styles.ts.Render(ts),
		mockStyle.Render(fmt.Sprintf("%-8s", mock)),
		l.styles.src.Render(fmt.Sprintf("%-15s", srcIP)),
		l.styles.arrow.Render("→"),
		l.styles.method.Render(fmt.Sprintf("%-6s", method)),
		l.styles.path.Render(path),
		statusStyle.Render(fmt.Sprintf("%3d", status)),
		l.styles.dim.Render(formatDuration(dur)),
	)
}

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Microsecond:
		return fmt.Sprintf("%dns", d.Nanoseconds())
	case d < time.Millisecond:
		return fmt.Sprintf("%dµs", d.Microseconds())
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	default:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

// Middleware returns net/http middleware that logs each request as the
// given mock name. The source IP is taken from RemoteAddr, with the first
// hop of X-Forwarded-For winning when present.
func (l *Logger) Middleware(mock string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &recordingWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)
			l.LogRequest(mock, sourceIP(r), r.Method, r.URL.RequestURI(), rw.status, time.Since(start))
		})
	}
}

// recordingWriter captures the response status code for the middleware.
type recordingWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *recordingWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.status = code
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *recordingWriter) Write(b []byte) (int, error) {
	w.wroteHeader = true
	return w.ResponseWriter.Write(b)
}

// sourceIP picks the most-trustworthy source address from the request: the
// first comma-separated entry in X-Forwarded-For, falling back to the host
// portion of RemoteAddr.
func sourceIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx > 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
