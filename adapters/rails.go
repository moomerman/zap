package adapters

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/moomerman/phx-dev/multiproxy"
	"github.com/puma/puma-dev/linebuffer"
	"github.com/vektra/errors"
)

// RailsAdapter holds the state for the application
type RailsAdapter struct {
	Host string
	Dir  string
	Port string

	cmd       *exec.Cmd
	proxy     *multiproxy.MultiProxy
	stdout    io.Reader
	log       linebuffer.LineBuffer
	readyChan chan struct{}
}

// CreateRailsAdapter creates a new rails adapter
func CreateRailsAdapter(host, dir string) (Adapter, error) {
	return &RailsAdapter{
		Host:      host,
		Dir:       dir,
		readyChan: make(chan struct{}),
	}, nil
}

const railsShellCommand = `exec bash -c '
exec bin/rails s -p %s'
`

// Start starts the application
func (a *RailsAdapter) Start() error {
	port, err := findAvailablePort()
	if err != nil {
		return errors.Context(err, "couldn't find available port")
	}

	a.Port = port

	if err := a.startApplication(railsShellCommand); err != nil {
		return errors.Context(err, "could not start application")
	}

	a.proxy = multiproxy.NewProxy("http://127.0.0.1:"+a.Port, a.Host)

	go a.tail()

	if err := a.wait(); err != nil {
		return errors.Context(err, "waiting for applicaftion to start")
	}

	return nil
}

// Stop stops the application
func (a *RailsAdapter) Stop() error {
	err := a.cmd.Process.Kill()
	if err != nil {
		fmt.Printf("! Error trying to stop %s: %s", a.Host, err)
		return err
	}

	a.cmd.Wait()

	fmt.Printf("* App '%s' shutdown and cleaned up\n", a.Host)
	return nil
}

// Command returns the command used to stat the application
func (a *RailsAdapter) Command() *exec.Cmd {
	return a.cmd
}

// WriteLog doesn't do anything
func (a *RailsAdapter) WriteLog(w io.Writer) {}

// ServeHTTP implements the http.Handler interface
func (a *RailsAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[proxy]", fullURL(r), "->", a.proxy.URL)
	a.proxy.Proxy(w, r)
}

func (a *RailsAdapter) startApplication(command string) error {
	shell := os.Getenv("SHELL")

	cmd := exec.Command(shell, "-l", "-i", "-c",
		fmt.Sprintf(command, a.Port))

	cmd.Dir = a.Dir

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("PHX_PORT=%s", a.Port))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	a.stdout = stdout
	cmd.Stderr = cmd.Stdout

	if err = cmd.Start(); err != nil {
		return errors.Context(err, "starting app")
	}

	a.cmd = cmd
	return nil
}

func (a *RailsAdapter) tail() error {
	c := make(chan error)

	go func() {
		r := bufio.NewReader(a.stdout)

		for {
			line, err := r.ReadString('\n')
			if line != "" {
				a.log.Append(line)
				fmt.Fprintf(os.Stdout, "  [app] %s:%s[%d]: %s", a.Host, a.Port, a.cmd.Process.Pid, line)

				ready, _ := regexp.Compile("Listening on tcp") // TODO: also grep for the host/port
				if ready.MatchString(line) {
					close(a.readyChan)
				}
			}

			if err != nil {
				c <- err
				return
			}
		}
	}()

	var err error

	select {
	case err = <-c:
		err = errors.Context(err, "stdout/stderr closed")
	}

	fmt.Println("  [app] stopping app", err)
	a.Stop()

	return err
}

func (a *RailsAdapter) wait() error {
	select {
	case <-a.readyChan:
		fmt.Println("[app] app ready", a.Host)
		return nil
	case <-time.After(time.Second * 30):
		close(a.readyChan)
		return errors.New("time out waiting for app to start")
	}
}
