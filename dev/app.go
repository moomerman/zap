package dev

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/moomerman/phx-dev/multiproxy"
	"github.com/puma/puma-dev/homedir"
	"github.com/vektra/errors"
)

const appsPath = "~/.phx-dev"

var apps map[string]*App
var lock sync.Mutex

type App struct {
	Host    string
	Port    string
	Link    string
	Dir     string
	Command *exec.Cmd

	proxy     *multiproxy.MultiProxy
	stdout    io.Reader
	lastUsed  time.Time
	readyChan chan struct{}
}

func (a *App) Start() error {
	path := homedir.MustExpand(appsPath) + "/" + a.Host
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}

	dir, err := os.Readlink(path)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		a.Link = path
		a.Dir = dir
	} else {
		return errors.New("unknown app")
	}

	return a.launch()
}

func (a *App) Stop(reason string, e error) error {
	fmt.Printf("! Stopping '%s' (%d) %s %s\n", a.Host, a.Command.Process.Pid, reason, e)
	lock.Lock()
	defer lock.Unlock()
	delete(apps, a.Host)

	err := a.Command.Process.Kill()
	if err != nil {
		fmt.Printf("! Error trying to stop %s: %s", a.Host, err)
		return err
	}

	a.Command.Wait()

	fmt.Printf("* App '%s' shutdown and cleaned up\n", a.Host)
	return nil
}

const executionShell = `exec bash -c '
cd %s
exec mix do deps.get, phx.server'
`

func (a *App) launch() error {
	shell := os.Getenv("SHELL")

	port, err := findAvailablePort()
	if err != nil {
		return errors.Context(err, "couldn't find available port")
	}

	a.Port = port

	cmd := exec.Command(shell, "-l", "-i", "-c",
		fmt.Sprintf(executionShell, a.Dir))

	cmd.Dir = a.Dir

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env,
		fmt.Sprintf("PHX_PORT=%s", a.Port),
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	a.stdout = stdout
	cmd.Stderr = cmd.Stdout

	err = cmd.Start()
	if err != nil {
		return errors.Context(err, "starting app")
	}

	a.Command = cmd
	a.proxy = multiproxy.NewProxy("http://127.0.0.1:"+a.Port, a.Host)

	go a.tail()

	err = a.wait()
	if err != nil {
		return errors.Context(err, "failed waiting for app to start")
	}

	go a.idleMonitor()

	return nil
}

func (a *App) tail() error {
	c := make(chan error)

	go func() {
		r := bufio.NewReader(a.stdout)

		for {
			line, err := r.ReadString('\n')
			if line != "" {
				fmt.Fprintf(os.Stdout, "  [app] %s:%s[%d]: %s", a.Host, a.Port, a.Command.Process.Pid, line)

				mustRestart, _ := regexp.Compile("You must restart your server")
				if mustRestart.MatchString(line) {
					c <- errors.New("Restart required")
					return
				}

				ready, _ := regexp.Compile("Running .*.Endpoint") // TODO: also grep for the port
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

	a.Stop("see error", err)

	return err
}

func (a *App) idleMonitor() error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if a.idle() {
				a.Stop("app is idle", nil)
				return nil
			}
		}
	}

	return nil
}

func (a *App) idle() bool {
	diff := time.Since(a.lastUsed)
	if diff > 60*60*time.Second {
		lock.Lock()
		defer lock.Unlock()
		return true
	}

	return false
}

func findAppForHost(host string) (*App, error) {
	lock.Lock()
	defer lock.Unlock()

	hostParts := strings.Split(host, ":")
	host = hostParts[0]

	if apps == nil {
		apps = make(map[string]*App)
	}

	app := apps[host]
	if app != nil {
		app.lastUsed = time.Now()
		return app, nil
	}

	fmt.Println("[app] attempting to start app for host", host)

	app = &App{
		Host:      host,
		readyChan: make(chan struct{}),
	}

	err := app.Start()
	if err != nil {
		fmt.Println("[app] error starting app for host", host, err)
		return nil, err
	}

	fmt.Println("[app] created app for host", host, app.Port)
	// TODO: apps should be keyed by Dir not host as you might have multiple
	// hosts pointing to the same app
	apps[host] = app
	return app, nil
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

func (a *App) wait() error {
	select {
	case <-a.readyChan:
		fmt.Println("[app] app ready", a.Host)
		a.lastUsed = time.Now()
		return nil
	}
}
