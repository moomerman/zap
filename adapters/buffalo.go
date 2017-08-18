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

	"github.com/moomerman/zap/multiproxy"
	"github.com/puma/puma-dev/linebuffer"
	"github.com/vektra/errors"
)

// BuffaloAdapter holds the state for the application
type BuffaloAdapter struct {
	Host string
	Dir  string
	Port string

	cmd       *exec.Cmd
	proxy     *multiproxy.MultiProxy
	stdout    io.Reader
	log       linebuffer.LineBuffer
	readyChan chan struct{}
}

// CreateBuffaloAdapter creates a new buffalo adapter
func CreateBuffaloAdapter(host, dir string) (Adapter, error) {
	return &BuffaloAdapter{
		Host:      host,
		Dir:       dir,
		readyChan: make(chan struct{}),
	}, nil
}

const buffaloShellCommand = `exec bash -c '
exec buffalo dev'
`

// Start starts the application
func (a *BuffaloAdapter) Start() error {
	port, err := findAvailablePort()
	if err != nil {
		return errors.Context(err, "couldn't find available port")
	}

	a.Port = port

	if err := a.startApplication(buffaloShellCommand); err != nil {
		return errors.Context(err, "could not start application")
	}

	a.proxy = multiproxy.NewProxy("http://127.0.0.1:"+a.Port, a.Host)

	go a.tail()

	if err := a.wait(); err != nil {
		return errors.Context(err, "waiting for application to start")
	}

	return nil
}

// Stop stops the application
func (a *BuffaloAdapter) Stop() error {
	// TODO: use a lock so only one goroutine can try and stop at one time?
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
func (a *BuffaloAdapter) Command() *exec.Cmd {
	return a.cmd
}

// WriteLog writes out the application log to the given writer
func (a *BuffaloAdapter) WriteLog(w io.Writer) {
	a.log.WriteTo(w)
}

// ServeHTTP implements the http.Handler interface
func (a *BuffaloAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[proxy]", fullURL(r), "->", a.proxy.URL)
	a.proxy.Proxy(w, r)
}

func (a *BuffaloAdapter) startApplication(command string) error {
	shell := os.Getenv("SHELL")

	cmd := exec.Command(shell, "-l", "-i", "-c",
		fmt.Sprintf(command, a.Port))

	cmd.Dir = a.Dir

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("PORT=%s", a.Port))

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

func (a *BuffaloAdapter) tail() error {
	c := make(chan error)

	go func() {
		r := bufio.NewReader(a.stdout)

		for {
			line, err := r.ReadString('\n')
			if line != "" {
				a.log.Append(line)
				fmt.Fprintf(os.Stdout, "  [log] %s:%s[%d]: %s", a.Host, a.Port, a.cmd.Process.Pid, line)

				ready, _ := regexp.Compile("Starting application at") // TODO: also grep for the host/port
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

// replace tail with a generic function that just waits until the http port
// is open and then closes the readyChan so it can be shared
// see https://github.com/puma/puma-dev/blob/master/dev/app.go#L318

func (a *BuffaloAdapter) wait() error {
	select {
	case <-a.readyChan:
		fmt.Println("[app] app ready", a.Host)
		return nil
	case <-time.After(time.Second * 30):
		close(a.readyChan)
		return errors.New("time out waiting for app to start")
	}
}
