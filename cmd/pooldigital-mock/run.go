package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/spf13/pflag"

	"github.com/yannicschroeer/pooldigital-mock/internal/logx"
	"github.com/yannicschroeer/pooldigital-mock/internal/proconip"
	"github.com/yannicschroeer/pooldigital-mock/internal/violet"
)

const (
	defaultHost         = "0.0.0.0"
	defaultProconipPort = 8080
	defaultVioletPort   = 8180
	defaultUser         = "admin"
	defaultPass         = "admin"
	shutdownTimeout     = 5 * time.Second
)

type options struct {
	host         string
	proconipPort int
	violetPort   int
	proconipUser string
	proconipPass string
	violetUser   string
	violetPass   string
	noColor      bool
	quiet        bool
	showVersion  bool
}

// run is the testable entry point: stateless beyond what's passed in.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	opts, helpRequested, err := parseFlags(args, stderr)
	if err != nil {
		return err
	}
	if helpRequested {
		return nil
	}
	if opts.showVersion {
		_, _ = fmt.Fprintln(stdout, "pooldigital-mock", version)
		return nil
	}

	logger := logx.New(logx.Options{Out: stdout, NoColor: opts.noColor, Quiet: opts.quiet})

	procState := proconip.New(proconip.Config{})
	violetSeed, err := violet.LoadSeed()
	if err != nil {
		return fmt.Errorf("load violet seed: %w", err)
	}
	violetState := violet.New(violet.Config{Seed: violetSeed})

	procHandler := proconip.NewHandler(proconip.ServerConfig{
		State: procState, Username: opts.proconipUser, Password: opts.proconipPass,
	})
	violetHandler := violet.NewHandler(violet.ServerConfig{
		State: violetState, Username: opts.violetUser, Password: opts.violetPass,
	})

	procAddr := net.JoinHostPort(opts.host, strconv.Itoa(opts.proconipPort))
	violetAddr := net.JoinHostPort(opts.host, strconv.Itoa(opts.violetPort))

	procSrv := &http.Server{
		Addr:              procAddr,
		Handler:           logger.Middleware("proconip")(procHandler),
		ReadHeaderTimeout: 10 * time.Second,
	}
	violetSrv := &http.Server{
		Addr:              violetAddr,
		Handler:           logger.Middleware("violet")(violetHandler),
		ReadHeaderTimeout: 10 * time.Second,
	}

	logger.Banner("pooldigital-mock", []logx.BannerLine{
		{Name: "proconip", URL: displayURL(opts.host, opts.proconipPort)},
		{Name: "violet", URL: displayURL(opts.host, opts.violetPort)},
	})

	errCh := make(chan error, 2)
	go func() {
		if err := procSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("proconip server: %w", err)
			return
		}
		errCh <- nil
	}()
	go func() {
		if err := violetSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("violet server: %w", err)
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutting down…")
	case err := <-errCh:
		if err != nil {
			_ = procSrv.Close()
			_ = violetSrv.Close()
			return err
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	_ = procSrv.Shutdown(shutdownCtx)
	_ = violetSrv.Shutdown(shutdownCtx)
	return nil
}

func parseFlags(args []string, errOut io.Writer) (options, bool, error) {
	fs := pflag.NewFlagSet("pooldigital-mock", pflag.ContinueOnError)
	fs.SetOutput(errOut)

	opts := options{
		host:         envOrDefault("HOST", defaultHost),
		proconipPort: envIntOrDefault("PROCONIP_MOCK_PORT", defaultProconipPort),
		violetPort:   envIntOrDefault("MYVIOLET_MOCK_PORT", defaultVioletPort),
		proconipUser: envOrDefault("PROCONIP_MOCK_USER", defaultUser),
		proconipPass: envOrDefault("PROCONIP_MOCK_PASS", defaultPass),
		violetUser:   envOrDefault("MYVIOLET_MOCK_USER", defaultUser),
		violetPass:   envOrDefault("MYVIOLET_MOCK_PASS", defaultPass),
	}

	fs.StringVar(&opts.host, "host", opts.host, "bind address")
	fs.IntVar(&opts.proconipPort, "proconip-port", opts.proconipPort, "proconip port")
	fs.IntVar(&opts.violetPort, "violet-port", opts.violetPort, "violet port")
	fs.StringVar(&opts.proconipUser, "proconip-user", opts.proconipUser, "proconip basic-auth user")
	fs.StringVar(&opts.proconipPass, "proconip-pass", opts.proconipPass, "proconip basic-auth pass")
	fs.StringVar(&opts.violetUser, "violet-user", opts.violetUser, "violet basic-auth user")
	fs.StringVar(&opts.violetPass, "violet-pass", opts.violetPass, "violet basic-auth pass")
	fs.BoolVar(&opts.noColor, "no-color", os.Getenv("NO_COLOR") != "", "disable ANSI colors")
	fs.BoolVar(&opts.quiet, "quiet", false, "suppress per-request logs")
	fs.BoolVar(&opts.showVersion, "version", false, "print version and exit")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			return opts, true, nil
		}
		return opts, false, err
	}
	return opts, false, nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOrDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return def
}

func displayURL(host string, port int) string {
	displayHost := host
	if host == "0.0.0.0" || host == "::" || host == "" {
		displayHost = "localhost"
	}
	return fmt.Sprintf("http://%s:%d", displayHost, port)
}
