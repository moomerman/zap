package adapters

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
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
	State           Status
	BootLog         string
	Pid             int

	cmd        *exec.Cmd
	proxy      *multiproxy.MultiProxy
	stdout     io.Reader
	log        linebuffer.LineBuffer
	cancelChan chan struct{}
}

// Start starts the application
func (a *AppProxyAdapter) Start() error {
	a.State = StatusStarting
	log.Println("[app]", a.Host, "starting")
	return a.start()
}

// Stop stops the application
func (a *AppProxyAdapter) Stop() error {
	a.State = StatusStopping
	log.Println("[app]", a.Host, "stopping")
	return a.stop()
}

// Restart restarts the adapter
func (a *AppProxyAdapter) Restart(reason error) error {
	a.State = StatusRestarting
	log.Println("[app]", a.Host, "restarting", reason)

	if err := a.stop(); a != nil {
		log.Printf("[app] %s error trying to stop on restart: %s", a.Host, err)
		a.State = StatusError
		return err
	}

	// if err := a.start(); a != nil {
	// 	log.Printf("[app] error trying to start on restart %s: %s", a.Host, err)
	// 	a.State = StatusError
	// 	return err
	// }

	return nil
}

// Status returns the status of the adapter
func (a *AppProxyAdapter) Status() Status {
	return a.State
}

// Command returns the command used to start the application
func (a *AppProxyAdapter) Command() *exec.Cmd {
	return a.cmd
}

// WriteLog writes the log to the given writer
func (a *AppProxyAdapter) WriteLog(w io.Writer) {
	a.log.WriteTo(w)
}

// ServeHTTP implements the http.Handler interface
func (a *AppProxyAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("[proxy]", fullURL(r), "->", a.proxy.URL)
	a.proxy.Proxy(w, r)
}

func (a *AppProxyAdapter) start() error {
	a.cancelChan = make(chan struct{})

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

func (a *AppProxyAdapter) stop() error {
	// TODO: use a lock so only one goroutine can try and stop at one time?
	err := a.cmd.Process.Kill()
	if err != nil {
		fmt.Printf("[app] error trying to stop %s: %s", a.Host, err)
		return err
	}

	a.cmd.Wait()

	log.Println("[app]", a.Host, "shutdown and cleaned up", err)
	a.State = StatusStopped
	a.Pid = 0

	return nil
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

	a.Pid = cmd.Process.Pid
	a.cmd = cmd
	return nil
}

func (a *AppProxyAdapter) tail() {
	c := make(chan error)

	go func() {
		r := bufio.NewReader(a.stdout)

		for {
			line, err := r.ReadString('\n')
			if line != "" {
				a.log.Append(line)
				fmt.Fprintf(os.Stdout, "  [log] %s:%s[%d]: %s", a.Host, a.Port, a.cmd.Process.Pid, line)

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
		log.Println("[app]", a.Host, "error tailing log", err)
		err = errors.Context(err, "stdout/stderr closed")
	}

}

func (a *AppProxyAdapter) checkPort() {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-a.cancelChan:
			log.Println("[app]", a.Host, "checkPort cancelChan closed")
			return
		case <-ticker.C:
			c, err := net.Dial("tcp", ":"+a.Port)
			if err == nil {
				log.Println("[app]", a.Host, "checkPort success")
				buf := bytes.NewBufferString("")
				a.WriteLog(buf)
				a.BootLog = buf.String()
				a.State = StatusRunning
				c.Close()
				return
			}
		case <-time.After(time.Second * 30):
			log.Println("[app]", a.Host, "checkPort timeout")
			a.State = StatusError // TODO: log this error as a timeout
			// return errors.New("time out waiting for app to start")
			return
		}
	}
}
