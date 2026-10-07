package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sync"
	"time"

	zadapter "github.com/moomerman/zap/adapter"
)

const (
	// bootTimeout is how long an app has to start listening on its port
	bootTimeout = 60 * time.Second
	// stopTimeout is how long an app has to exit after SIGTERM before SIGKILL
	stopTimeout = 5 * time.Second
	// waitDelay bounds how long Wait waits for output after the process exits,
	// in case a grandchild that escaped the process group holds the pipe open
	waitDelay = 2 * time.Second
)

// Config holds the server configuration
type Config struct {
	Name            string
	Scheme          string
	Host            string
	Dir             string
	EnvPortName     string
	ShellCommand    string
	RestartPatterns []*regexp.Regexp

	// OnStatus is called when the server finishes booting, fails or exits
	OnStatus zadapter.StatusFunc
	// OnLog is called with each line of output, eg. to write it to a file
	OnLog func(line string)
}

// New returns a new server adapter
func New(config *Config) zadapter.Adapter {
	return &adapter{
		config: *config,
		state:  zadapter.StatusStopped,
	}
}

type adapter struct {
	config Config
	log    lineBuffer

	mu      sync.Mutex
	state   zadapter.Status
	err     error
	port    string
	command string
	bootLog string
	proxy   *httputil.ReverseProxy
	run     *run
}

// run is a single execution of the server process
type run struct {
	cmd      *exec.Cmd
	pid      int
	port     string
	out      *lineWriter
	done     chan struct{} // closed once the process has exited
	stopping chan struct{} // closed when a stop is requested
	stopOnce sync.Once
}

func (r *run) stop() {
	r.stopOnce.Do(func() { close(r.stopping) })
}

// Start starts the application
func (a *adapter) Start() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch a.state {
	case zadapter.StatusStarting, zadapter.StatusRunning, zadapter.StatusStopping:
		return nil
	}

	log.Println("[app]", a.config.Host, "START")
	a.err = nil
	a.bootLog = ""
	a.setState(zadapter.StatusStarting)

	port, err := findAvailablePort()
	if err != nil {
		return a.fail(fmt.Errorf("couldn't find available port: %w", err))
	}
	a.port = port

	target, err := url.Parse(a.config.Scheme + "://127.0.0.1:" + port)
	if err != nil {
		return a.fail(err)
	}
	// an empty hostname keeps the Host header of each incoming request, so
	// apps that serve several hosts share one proxy
	a.proxy = zadapter.NewReverseProxy(target, "")

	r, err := a.startProcess()
	if err != nil {
		return a.fail(fmt.Errorf("could not start application: %w", err))
	}
	a.run = r

	go a.wait(r)
	go a.checkPort(r)

	return nil
}

// Stop stops the application, sending SIGTERM to its process group and
// escalating to SIGKILL if it hasn't exited within stopTimeout
func (a *adapter) Stop(reason error) error {
	a.mu.Lock()
	r := a.run
	if r == nil || a.state == zadapter.StatusStopped {
		a.setState(zadapter.StatusStopped)
		a.mu.Unlock()
		return nil
	}
	if a.state != zadapter.StatusStopping {
		log.Println("[app]", a.config.Host, "STOP", reason)
		a.setState(zadapter.StatusStopping)
	}
	a.mu.Unlock()

	r.stop()

	select {
	case <-r.done:
	default:
		if err := terminate(r.cmd); err != nil {
			log.Println("[app]", a.config.Host, "error sending SIGTERM", err)
		}
	}

	select {
	case <-r.done:
	case <-time.After(stopTimeout):
		log.Println("[app]", a.config.Host, "did not exit, sending SIGKILL")
		if err := kill(r.cmd); err != nil {
			log.Println("[app]", a.config.Host, "error sending SIGKILL", err)
		}
		<-r.done
	}

	// the process may have exited before the stop was requested, in which
	// case wait has already run and left the state alone
	a.mu.Lock()
	if a.run == r && a.state == zadapter.StatusStopping {
		a.setState(zadapter.StatusStopped)
	}
	a.mu.Unlock()

	return nil
}

// Status returns the status of the adapter
func (a *adapter) Status() zadapter.Status {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.state
}

// Snapshot returns the current state of the adapter
func (a *adapter) Snapshot() zadapter.Snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()

	s := zadapter.Snapshot{
		Name:    a.config.Name,
		Status:  a.state,
		Port:    a.port,
		Command: a.command,
		BootLog: a.bootLog,
	}
	if a.run != nil && a.state != zadapter.StatusStopped {
		s.Pid = a.run.pid
	}
	if a.err != nil {
		s.Error = a.err.Error()
	}
	return s
}

// WriteLog writes the log to the given writer
func (a *adapter) WriteLog(w io.Writer) {
	a.log.WriteTo(w)
}

// ServeHTTP implements the http.Handler interface
func (a *adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	proxy, port := a.proxy, a.port
	a.mu.Unlock()

	if proxy == nil {
		http.Error(w, "502 Bad Gateway", http.StatusBadGateway)
		return
	}

	log.Println("[proxy]", zadapter.FullURL(r), "->", port)
	proxy.ServeHTTP(w, r)
}

// fail records a start failure. The caller must hold mu.
func (a *adapter) fail(err error) error {
	log.Println("[app]", a.config.Host, "ERROR", err)
	a.err = err
	a.setState(zadapter.StatusError)
	return err
}

// setState changes the state and notifies the observer. The caller must hold
// mu, which keeps notifications in the same order as the transitions.
func (a *adapter) setState(state zadapter.Status) {
	if a.state == state {
		return
	}
	a.state = state
	if a.config.OnStatus != nil {
		a.config.OnStatus(state, a.err)
	}
}

func (a *adapter) startProcess() (*run, error) {
	shell := os.Getenv("SHELL")

	command := fmt.Sprintf(a.config.ShellCommand, a.port, a.config.Host)
	a.command = command
	log.Println("[app] command:", command)

	cmd := exec.Command(shell, "-l", "-i", "-c", command)
	cmd.Dir = a.config.Dir
	setProcessGroup(cmd)

	cmd.Env = os.Environ()
	if a.config.EnvPortName != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", a.config.EnvPortName, a.port))
	}

	appEnv, err := readEnvFile(cmd.Dir)
	if err != nil {
		log.Println("[app]", a.config.Host, "ERROR", "couldn't read env file", err)
	}

	for _, pair := range appEnv {
		log.Println("[app]", a.config.Host, "INFO", "added env var", sanitiseEnvPair(pair))
		cmd.Env = append(cmd.Env, pair)
	}

	r := &run{
		cmd:      cmd,
		port:     a.port,
		done:     make(chan struct{}),
		stopping: make(chan struct{}),
	}

	// Stdout and Stderr are the same writer, so exec calls Write from a
	// single goroutine
	r.out = &lineWriter{line: func(line string) { a.logLine(r, line) }}
	cmd.Stdout = r.out
	cmd.Stderr = r.out
	cmd.WaitDelay = waitDelay

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	r.pid = cmd.Process.Pid

	return r, nil
}

func (a *adapter) logLine(r *run, line string) {
	a.log.Append(line)

	if a.config.OnLog != nil {
		a.config.OnLog(line)
	}

	for _, pattern := range a.config.RestartPatterns {
		if pattern.MatchString(line) {
			// Stop waits for output to drain, so it can't run on this goroutine
			go a.Stop(errors.New("restart pattern matched"))
			return
		}
	}
}

// wait waits for the process to exit and records why it did
func (a *adapter) wait(r *run) {
	err := r.cmd.Wait()
	r.out.flush()
	close(r.done)

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.run != r {
		return
	}

	switch a.state {
	case zadapter.StatusStopping:
		log.Println("[app]", a.config.Host, "shutdown and cleaned up")
		a.setState(zadapter.StatusStopped)
	case zadapter.StatusStarting, zadapter.StatusRunning:
		if err == nil {
			err = errors.New("exited")
		}
		a.fail(fmt.Errorf("process exited unexpectedly: %w", err))
	}
}

func (a *adapter) checkPort(r *run) {
	ticker := time.NewTicker(250 * time.Millisecond)
	timeout := time.After(bootTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopping:
			return
		case <-r.done:
			return
		case <-ticker.C:
			c, err := net.Dial("tcp", ":"+r.port)
			if err != nil {
				continue
			}
			c.Close()

			buf := bytes.NewBufferString("")
			a.WriteLog(buf)

			a.mu.Lock()
			if a.run == r && a.state == zadapter.StatusStarting {
				log.Println("[app]", a.config.Host, "port", r.port, "is available")
				a.bootLog = buf.String()
				a.setState(zadapter.StatusRunning)
			}
			a.mu.Unlock()
			return
		case <-timeout:
			log.Println("[app]", a.config.Host, "timeout waiting for port", r.port)
			a.mu.Lock()
			if a.run == r && a.state == zadapter.StatusStarting {
				a.fail(errors.New("timed out waiting for the app to listen on its port"))
			}
			a.mu.Unlock()
			// the process is still running, so clean it up
			go a.Stop(errors.New("boot timeout"))
			return
		}
	}
}

func findAvailablePort() (string, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", err
	}
	l.Close()

	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return "", err
	}

	return port, nil
}
