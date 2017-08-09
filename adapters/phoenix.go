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

// PhoenixAdapter holds the state for the application
type PhoenixAdapter struct {
	Host string
	Dir  string
	Port string

	cmd       *exec.Cmd
	proxy     *multiproxy.MultiProxy
	stdout    io.Reader
	log       linebuffer.LineBuffer
	readyChan chan struct{}
}

// CreatePhoenixAdapter creates a new phoenix adapter
func CreatePhoenixAdapter(host, dir string) (Adapter, error) {
	return &PhoenixAdapter{
		Host:      host,
		Dir:       dir,
		readyChan: make(chan struct{}),
	}, nil
}

const phoenixShellCommand = `exec bash -c '
cd %s
exec mix do deps.get, phx.server'
`

// Start starts a phoenix application
func (d *PhoenixAdapter) Start() error {
	port, err := findAvailablePort()
	if err != nil {
		return errors.Context(err, "couldn't find available port")
	}

	d.Port = port

	if err := d.startApplication(phoenixShellCommand); err != nil {
		return errors.Context(err, "could not start application")
	}

	d.proxy = multiproxy.NewProxy("http://127.0.0.1:"+d.Port, d.Host)

	go d.tail()

	if err := d.wait(); err != nil {
		return errors.Context(err, "waiting for applicaftion to start")
	}

	return nil
}

// Stop stops a phoenix application
func (d *PhoenixAdapter) Stop() error {
	err := d.cmd.Process.Kill()
	if err != nil {
		fmt.Printf("! Error trying to stop %s: %s", d.Host, err)
		return err
	}

	d.cmd.Wait()

	fmt.Printf("* App '%s' shutdown and cleaned up\n", d.Host)
	return nil
}

// Command returns the command used to start the application
func (d *PhoenixAdapter) Command() *exec.Cmd {
	return d.cmd
}

// WriteLog writes out the application log to the given writer
func (d *PhoenixAdapter) WriteLog(w io.Writer) {
	d.log.WriteTo(w)
}

// ServeHTTP implements the http.Handler interface
func (d *PhoenixAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[proxy]", fullURL(r), "->", d.proxy.URL)
	d.proxy.Proxy(w, r)
}

func (d *PhoenixAdapter) startApplication(command string) error {
	shell := os.Getenv("SHELL")

	cmd := exec.Command(shell, "-l", "-i", "-c",
		fmt.Sprintf(command, d.Dir))

	cmd.Dir = d.Dir

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env,
		fmt.Sprintf("PHX_PORT=%s", d.Port),
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	d.stdout = stdout
	cmd.Stderr = cmd.Stdout

	if err = cmd.Start(); err != nil {
		return errors.Context(err, "starting app")
	}

	d.cmd = cmd
	return nil
}

func (d *PhoenixAdapter) tail() error {
	c := make(chan error)

	go func() {
		r := bufio.NewReader(d.stdout)

		for {
			line, err := r.ReadString('\n')
			if line != "" {
				d.log.Append(line)
				fmt.Fprintf(os.Stdout, "  [log] %s:%s[%d]: %s", d.Host, d.Port, d.cmd.Process.Pid, line)

				mustRestart, _ := regexp.Compile("You must restart your server")
				if mustRestart.MatchString(line) {
					c <- errors.New("Restart required")
					return
				}

				ready, _ := regexp.Compile("Running .*.Endpoint") // TODO: also grep for the port
				if ready.MatchString(line) {
					close(d.readyChan)
				}
			}

			// TODO: handle installing js assets?
			// [error] Could not start node watcher because script "/Users/richard/workspace/moocode/hoot/apps/web/assets/node_modules/brunch/bin/brunch" does not exist. Your Phoenix application is still running, however assets won't be compiled. You may fix this by running "cd assets && npm install"

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
	d.Stop()

	return err
}

func (d *PhoenixAdapter) wait() error {
	select {
	case <-d.readyChan:
		fmt.Println("[app] app ready", d.Host)
		return nil
	case <-time.After(time.Second * 10):
		close(d.readyChan)
		return errors.New("time out waiting for app to start")
	}
}
