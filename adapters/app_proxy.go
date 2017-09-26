package adapters

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/moomerman/zap/multiproxy"
	"github.com/puma/puma-dev/linebuffer"
	"github.com/vektra/errors"
)

// AppProxyAdapter holds the state for the application
type AppProxyAdapter struct {
	Host            string
	Dir             string
	Port            string
	ShellCommand    string
	EnvPortName     string
	RestartPatterns []*regexp.Regexp
	Cmd             *exec.Cmd
	State           Status

	proxy      *multiproxy.MultiProxy
	stdout     io.Reader
	log        linebuffer.LineBuffer
	cancelChan chan struct{}
}

// Start starts the application
func (a *AppProxyAdapter) Start() error {
	a.cancelChan = make(chan struct{})
	a.State = StatusStarting

	port, err := findAvailablePort()
	if err != nil {
		a.State = StatusError
		return errors.Context(err, "couldn't find available port")
	}

	a.Port = port

	if err := a.startApplication(a.ShellCommand); err != nil {
		a.State = StatusError
		return errors.Context(err, "could not start application")
	}

	a.proxy = multiproxy.NewProxy("http://127.0.0.1:"+a.Port, a.Host)

	go a.tail()
	go a.checkPort()

	return nil
}

// Stop stops the application
func (a *AppProxyAdapter) Stop() error {
	a.State = StatusStopping
	// TODO: use a lock so only one goroutine can try and stop at one time?
	err := a.Cmd.Process.Kill()
	if err != nil {
		fmt.Printf("  [app] error trying to stop %s: %s", a.Host, err)
		return err
	}

	a.Cmd.Wait()

	fmt.Println("  [app] shutdown and cleaned up", a.Host, err)
	a.State = StatusStopped
	return nil
}

// Restart restarts the adapter
func (a *AppProxyAdapter) Restart(reason error) error {
	a.State = StatusRestarting
	// TODO: lock
	// TODO: stop/cleanup
	err := a.Cmd.Process.Kill()
	if err != nil {
		fmt.Printf("  [app] error trying to stop on restart %s: %s", a.Host, err)
		a.State = StatusError
		return err
	}

	a.Cmd.Wait()

	a.Start()

	return nil
}

// Status returns the status of the adapter
func (a *AppProxyAdapter) Status() Status {
	return a.State
}

// Command returns the command used to start the application
func (a *AppProxyAdapter) Command() *exec.Cmd {
	return a.Cmd
}

// WriteLog writes the log to the given writer
func (a *AppProxyAdapter) WriteLog(w io.Writer) {
	a.log.WriteTo(w)
}

// ServeHTTP implements the http.Handler interface
func (a *AppProxyAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[proxy]", fullURL(r), "->", a.proxy.URL)
	a.proxy.Proxy(w, r)
}

func (a *AppProxyAdapter) startApplication(command string) error {
	shell := os.Getenv("SHELL")

	cmd := exec.Command(shell, "-l", "-i", "-c",
		fmt.Sprintf(command, a.Port, a.Host))

	cmd.Dir = a.Dir

	cmd.Env = os.Environ()
	if a.EnvPortName != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", a.EnvPortName, a.Port))
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	a.stdout = stdout
	cmd.Stderr = cmd.Stdout

	if err = cmd.Start(); err != nil {
		return errors.Context(err, "starting app")
	}

	a.Cmd = cmd
	return nil
}

func (a *AppProxyAdapter) tail() error {
	c := make(chan error)

	go func() {
		r := bufio.NewReader(a.stdout)

		for {
			line, err := r.ReadString('\n')
			if line != "" {
				a.log.Append(line)
				fmt.Fprintf(os.Stdout, "  [log] %s:%s[%d]: %s", a.Host, a.Port, a.Cmd.Process.Pid, line)

				for _, pattern := range a.RestartPatterns {
					if pattern.MatchString(line) {
						a.Restart(errors.New("Restart pattern matched"))
						return
					}
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
		a.State = StatusError
		close(a.cancelChan)
		fmt.Println("  [app] error tailing log", a.Host, err)
		err = errors.Context(err, "stdout/stderr closed")
	}

	return err
}

func (a *AppProxyAdapter) checkPort() {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-a.cancelChan:
			fmt.Println("  [app] checkPort cancelChan closed", a.Host)
			return
		case <-ticker.C:
			c, err := net.Dial("tcp", ":"+a.Port)
			if err == nil {
				c.Close()
				a.State = StatusRunning
				return
			}
		case <-time.After(time.Second * 30):
			fmt.Println("  [app] checkPort timeout", a.Host)
			a.State = StatusError // TODO: log this error as a timeout
			// return errors.New("time out waiting for app to start")
			return
		}
	}
}
