package logx

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBannerContainsTitleAndLines(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Out: &buf, NoColor: true})
	l.Banner("pooldigital-mock", []BannerLine{
		{Name: "proconip", URL: "http://0.0.0.0:8080"},
		{Name: "violet", URL: "http://0.0.0.0:8180"},
	})
	got := buf.String()
	for _, want := range []string{"pooldigital-mock", "proconip", "http://0.0.0.0:8080", "violet", "http://0.0.0.0:8180"} {
		if !strings.Contains(got, want) {
			t.Errorf("banner missing %q\n--- output ---\n%s", want, got)
		}
	}
}

func TestLogRequestFormat(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Out: &buf, NoColor: true})
	l.LogRequest("proconip", "192.168.1.42", "GET", "/GetState.csv", 200, 3*time.Millisecond)
	got := buf.String()
	for _, want := range []string{"proconip", "192.168.1.42", "→", "GET", "/GetState.csv", "200", "3ms"} {
		if !strings.Contains(got, want) {
			t.Errorf("log line missing %q\n--- output ---\n%s", want, got)
		}
	}
}

func TestQuietSuppressesRequestLines(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Out: &buf, NoColor: true, Quiet: true})
	l.LogRequest("violet", "127.0.0.1", "GET", "/getReadings", 200, time.Millisecond)
	if buf.Len() != 0 {
		t.Errorf("quiet mode should suppress request lines, got %q", buf.String())
	}
}

func TestMiddlewareCapturesStatusAndPath(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Out: &buf, NoColor: true})
	mw := l.Middleware("violet")

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("i am a teapot"))
	}))

	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/brew?type=earlgrey")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_ = resp.Body.Close()

	got := buf.String()
	for _, want := range []string{"violet", "GET", "/brew?type=earlgrey", "418"} {
		if !strings.Contains(got, want) {
			t.Errorf("middleware output missing %q\n--- output ---\n%s", want, got)
		}
	}
}

func TestMiddlewareHonorsXForwardedFor(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Out: &buf, NoColor: true})
	mw := l.Middleware("proconip")

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	srv := httptest.NewServer(handler)
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("X-Forwarded-For", "10.0.0.7, 10.0.0.1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	_ = resp.Body.Close()

	if !strings.Contains(buf.String(), "10.0.0.7") {
		t.Errorf("expected first XFF hop in output, got %q", buf.String())
	}
}

func TestFormatDurationBuckets(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{500 * time.Nanosecond, "500ns"},
		{500 * time.Microsecond, "500µs"},
		{5 * time.Millisecond, "5ms"},
		{1500 * time.Millisecond, "1.50s"},
	}
	for _, c := range cases {
		if got := formatDuration(c.in); got != c.want {
			t.Errorf("formatDuration(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestInfoRespectsQuiet(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Out: &buf, NoColor: true, Quiet: true})
	l.Info("hello %s", "world")
	if buf.Len() != 0 {
		t.Errorf("Info should be silent when quiet, got %q", buf.String())
	}

	buf.Reset()
	l2 := New(Options{Out: &buf, NoColor: true})
	l2.Info("hello %s", "world")
	if !strings.Contains(buf.String(), "hello world") {
		t.Errorf("Info missing message, got %q", buf.String())
	}
}
