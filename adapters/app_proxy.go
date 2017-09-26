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
	"sync"
	"time"

	"github.com/moomerman/zap/multiproxy"
	"github.com/puma/puma-dev/linebuffer"
	"github.com/vektra/errors"
)

// AppProxyAdapter holds the state for the application
type AppProxyAdapter struct {
	sync.Mutex

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
	a.Lock()
	defer a.Unlock()
	if a.State == StatusStopping || a.State == StatusRunning {
		return nil
	}

	log.Println("[app]", a.Host, "START")
	return a.start()
}

// Stop stops the application
func (a *AppProxyAdapter) Stop(reason error) error {
	a.Lock()
	defer a.Unlock()
	if a.State == StatusStopping || a.State == StatusStopped {
		return nil
	}

	log.Println("[app]", a.Host, "STOP", reason)
	return a.stop()
}

// Status returns the status of the adapter
func (a *AppProxyAdapter) Status() Status {
	return a.State
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
	a.State = StatusStarting
	a.cancelChan = make(chan struct{})

	port, err := findAvailablePort()
	if err != nil {
		e := errors.Context(err, "couldn't find available port")
		a.Stop(e)
		return e
	}

	a.Port = port

	if err := a.startApplication(a.ShellCommand); err != nil {
		e := errors.Context(err, "could not start application")
		a.Stop(e)
		return e
	}

	a.proxy = multiproxy.NewProxy("http://127.0.0.1:"+a.Port, a.Host)

	go a.tail()
	go a.checkPort()

	return nil
}

func (a *AppProxyAdapter) stop() error {
	a.State = StatusStopping
	close(a.cancelChan)

	err := a.cmd.Process.Kill()
	if err != nil {
		log.Println("[app]", a.Host, "error trying to stop", err)
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

func (a *AppProxyAdapter) checkPort() {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-a.cancelChan:
			return
		case <-ticker.C:
			c, err := net.Dial("tcp", ":"+a.Port)
			if err == nil {
				defer c.Close()
				log.Println("[app]", a.Host, "port available")
				buf := bytes.NewBufferString("")
				a.WriteLog(buf)
				a.BootLog = buf.String()
				a.State = StatusRunning
				return
			}
		case <-time.After(time.Second * 30):
			log.Println("[app]", a.Host, "port timeout")
			a.Stop(errors.New("port timeout"))
			return
		}
	}
}
