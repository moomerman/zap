package ngrok

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"time"
)

// Tunnel represents an ngrok tunnel
type Tunnel struct {
	Host     string
	URL      string
	AdminURL string
	Pid      int
	Command  string

	cmd    *exec.Cmd
	stdout io.Reader
}

// StartTunnel starts a new ngrok tunnel for the given host and port
func StartTunnel(host string, port int) (*Tunnel, error) {

	ngrok, err := startTunnel(host, port)
	if err != nil {
		return nil, err
	}
	go ngrok.tail()
	ngrok.wait()

	return ngrok, nil
}

// Stop stops the ngrok instance
func (n *Tunnel) Stop() error {
	err := n.cmd.Process.Kill()
	if err != nil {
		log.Println("[ngrok]", n.Host, "error trying to stop", err)
		return err
	}

	n.cmd.Wait()
	log.Println("[ngrok]", "stopped")

	return nil
}

func (n *Tunnel) tail() {
	c := make(chan error)
	adminR := regexp.MustCompile(`obj=web addr=(127.0.0.1:\d+)`)
	urlR := regexp.MustCompile(`URL:(https://[a-z0-9]+.[a-z.]*ngrok.io) `)

	go func() {

		r := bufio.NewReader(n.stdout)
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				c <- err
				return
			}

			if line == "" {
				continue
			}

			// log.Print(line)

			match := adminR.FindStringSubmatch(line)
			if len(match) > 0 {
				n.AdminURL = match[len(match)-1]
				log.Println("[ngrok] admin url", n.AdminURL)
			}

			match = urlR.FindStringSubmatch(line)
			if len(match) > 0 {
				n.URL = match[len(match)-1]
				log.Println("[ngrok] tunnel url", n.URL)
			}
		}
	}()

	var err error
	select {
	case err = <-c:
		log.Println("[ngrok]", "exited", err)
	}

}

func (n *Tunnel) wait() {
	timeout := time.After(time.Second * 5)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if n.URL != "" && n.AdminURL != "" {
				log.Println("[ngrok]", "ready")
				return
			}
		case <-timeout:
			log.Println("[ngrok]", "timeout")
			n.Stop()
			return
		}
	}
}

func startTunnel(host string, port int) (*Tunnel, error) {
	// TODO: make the options configurable
	command := fmt.Sprintf("exec ngrok http --region=eu --host-header=%s --bind-tls=true --log-format=logfmt --log=stdout --log-level=debug %d", host, port)
	shell := os.Getenv("SHELL")
	cmd := exec.Command(shell, "-l", "-c", command)
	cmd.Env = os.Environ()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	cmd.Stderr = cmd.Stdout

	if err = cmd.Start(); err != nil {
		return nil, err
	}

	return &Tunnel{
		Host:    host,
		Command: command,
		Pid:     cmd.Process.Pid,
		cmd:     cmd,
		stdout:  stdout,
	}, nil

}
