// Command team-server runs the operator side of the C2 teaching kit. It accepts
// check-ins from Beacons, stores their results, and gives the operator a
// console.
package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/had-nu/Go-ServerC2-Payload/internal/agentstore"
	"github.com/had-nu/Go-ServerC2-Payload/internal/transport"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "team-server: %v\n", err)
		os.Exit(1)
	}
}

type config struct {
	addr      string
	stateDir  string
	useTLS    bool
	secretArg string
}

func run() error {
	var cfg config
	flag.StringVar(&cfg.addr, "addr", ":8443", "address that the server listens on")
	flag.StringVar(&cfg.stateDir, "state", defaultStateDir(), "directory for the secret and the certificate")
	flag.BoolVar(&cfg.useTLS, "tls", true, "serve HTTPS instead of HTTP")
	flag.StringVar(&cfg.secretArg, "secret-file", "", "path to the shared secret; the default is <state>/c2.key")
	flag.Parse()

	log := transport.NewLogger(os.Stderr)

	secretPath := cfg.secretArg
	if secretPath == "" {
		secretPath = filepath.Join(cfg.stateDir, "c2.key")
	}
	secret, err := transport.LoadSecret(secretPath)
	if err != nil {
		return err
	}
	log.Info("shared secret ready", "path", secretPath)

	store := agentstore.New()
	events := make(chan transport.Event, 64)
	srv := transport.NewServer(store, secret, log, events)

	httpSrv, err := buildHTTPServer(cfg, srv)
	if err != nil {
		return err
	}

	ln, err := listen(httpSrv, cfg.useTLS)
	if err != nil {
		return err
	}
	defer ln.Close()

	printBanner(cfg, ln.Addr().String(), secretPath)

	console := newConsole(os.Stdout, store, events)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	done := make(chan error, 1)
	go func() { done <- console.run() }()

	var consoleErr error
	select {
	case s := <-sig:
		log.Info("shutting down", "signal", s.String())
	case consoleErr = <-done:
	}

	shutdownErr := transport.Shutdown(httpSrv)
	console.stop()
	if consoleErr != nil && !errors.Is(consoleErr, errConsoleExit) {
		return consoleErr
	}
	if shutdownErr != nil {
		return fmt.Errorf("shutdown: %w", shutdownErr)
	}
	return nil
}

func buildHTTPServer(cfg config, h http.Handler) (*http.Server, error) {
	if !cfg.useTLS {
		return transport.PlainHTTPServer(cfg.addr, h), nil
	}
	cert, err := transport.LoadOrCreateCert(cfg.stateDir, nil)
	if err != nil {
		return nil, err
	}
	return transport.HTTPServer(cfg.addr, h, cert), nil
}

func listen(srv *http.Server, useTLS bool) (net.Listener, error) {
	if useTLS {
		return transport.ListenTLS(srv)
	}
	return transport.Listen(srv)
}

func printBanner(cfg config, addr, secretPath string) {
	scheme := "http"
	if cfg.useTLS {
		scheme = "https"
	}
	fmt.Printf("[*] team server listening on %s://%s\n", scheme, addr)
	if cfg.useTLS {
		certPath, _ := transport.CertPaths(cfg.stateDir)
		fmt.Printf("[*] certificate: %s\n", certPath)
	}
	fmt.Printf("[*] shared secret: %s\n\n", secretPath)
}

func defaultStateDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "go-serverc2-payload")
	}
	return filepath.Join(home, ".local", "share", "go-serverc2-payload")
}
