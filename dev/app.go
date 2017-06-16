package dev

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
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

	proxy  *multiproxy.MultiProxy
	stdout io.Reader
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

const executionShell = `exec bash -c '
cd %s
exec mix phx.server'
`

func (a *App) launch() error {
	shell := os.Getenv("SHELL")
	addr := findAvailableAddr()
	_, port, err := net.SplitHostPort(addr)
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

	time.Sleep(5 * time.Second)

	go a.tail()

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

	a.Command.Process.Kill()
	a.Command.Wait()

	lock.Lock()
	defer lock.Unlock()
	delete(apps, a.Host)

	fmt.Printf("* App '%s' shutdown and cleaned up\n", a.Host)

	return err
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
		return app, nil
	}

	fmt.Println("[app] attempting to start app for host", host)

	app = &App{
		Host: host,
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

func findAvailableAddr() string {
	l, _ := net.Listen("tcp", ":0")
	l.Close()
	return l.Addr().String()
}
