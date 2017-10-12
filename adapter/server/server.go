package server

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
	"sync"
	"time"

	"github.com/moomerman/zap/adapter"
	"github.com/moomerman/zap/proxy"
	"github.com/puma/puma-dev/linebuffer"
	"github.com/vektra/errors"
)

// Adapter holds the state for the application
type Adapter struct {
	sync.Mutex

	Name            string
	Host            string
	Dir             string
	Port            string
	Command         string
	EnvPortName     string           `json:",omitempty"`
	RestartPatterns []*regexp.Regexp `json:",omitempty"`
	BootLog         string
	Pid             int
	ShellCommand    string

	state      adapter.Status
	cmd        *exec.Cmd
	proxy      *proxy.MultiProxy
	stdout     io.Reader
	log        linebuffer.LineBuffer
	cancelChan chan struct{}
}

// Start starts the application
func (a *Adapter) Start() error {
	a.Lock()
	defer a.Unlock()
	if a.state == adapter.StatusStopping || a.state == adapter.StatusRunning {
		return nil
	}

	log.Println("[app]", a.Host, "START")
	return a.start()
}

// Stop stops the application
func (a *Adapter) Stop(reason error) error {
	a.Lock()
	defer a.Unlock()
	if a.state == adapter.StatusStopping || a.state == adapter.StatusStopped {
		return nil
	}

	log.Println("[app]", a.Host, "STOP", reason)
	return a.stop()
}

// Status returns the status of the adapter
func (a *Adapter) Status() adapter.Status {
	a.Lock()
	defer a.Unlock()
	return a.state
}

// WriteLog writes the log to the given writer
func (a *Adapter) WriteLog(w io.Writer) {
	a.log.WriteTo(w)
}

// ServeHTTP implements the http.Handler interface
func (a *Adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("[proxy]", adapter.FullURL(r), "->", a.proxy.URL)
	a.proxy.Proxy(w, r)
}

func (a *Adapter) start() error {
	a.state = adapter.StatusStarting
	a.cancelChan = make(chan struct{})

	port, err := adapter.FindAvailablePort()
	if err != nil {
		e := errors.Context(err, "couldn't find available port")
		a.error(e)
		return e
	}

	a.Port = port

	if err := a.startApplication(a.ShellCommand); err != nil {
		e := errors.Context(err, "could not start application")
		a.error(e)
		return e
	}

	proxy, err := proxy.NewProxy("http://127.0.0.1:"+a.Port, a.Host)
	if err != nil {
		return err
	}

	a.proxy = proxy
	go a.tail()
	go a.checkPort()

	return nil
}

func (a *Adapter) stop() error {
	a.state = adapter.StatusStopping
	close(a.cancelChan)

	err := a.cmd.Process.Kill()
	if err != nil {
		log.Println("[app]", a.Host, "error trying to stop", err)
		return err
	}

	a.cmd.Wait()

	log.Println("[app]", a.Host, "shutdown and cleaned up")
	a.changeState(adapter.StatusStopped)
	a.Pid = 0

	return nil
}

func (a *Adapter) error(err error) error {
	if a.state == adapter.StatusStopping || a.state == adapter.StatusStopped {
		return nil
	}

	log.Println("[app]", a.Host, "ERROR", err)

	if err := a.stop(); err != nil {
		return err
	}

	a.changeState(adapter.StatusError)
	return nil
}

func (a *Adapter) startApplication(command string) error {
	shell := os.Getenv("SHELL")

	command = fmt.Sprintf(command, a.Port, a.Host)
	a.Command = command

	cmd := exec.Command(shell, "-l", "-c", command)
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

func (a *Adapter) tail() {
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
						a.Stop(errors.New("Restart pattern matched"))
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
		a.Stop(errors.Context(err, "stdout/stderr closed"))
	}

}

func (a *Adapter) checkPort() {
	ticker := time.NewTicker(250 * time.Millisecond)
	timeout := time.After(time.Second * 30)
	defer ticker.Stop()

	for {
		select {
		case <-a.cancelChan:
			return
		case <-ticker.C:
			c, err := net.Dial("tcp", ":"+a.Port)
			if err == nil {
				defer c.Close()
				log.Println("[app]", a.Host, "check port available")
				buf := bytes.NewBufferString("")
				a.WriteLog(buf)
				a.BootLog = buf.String()
				a.changeState(adapter.StatusRunning)
				return
			}
		case <-timeout:
			log.Println("[app]", a.Host, "check port timeout")
			a.error(errors.New("check port timeout"))
			return
		}
	}
}

func (a *Adapter) changeState(state adapter.Status) {
	a.Lock()
	defer a.Unlock()
	a.state = state
}
