// Command zap shows and controls the apps that zapd is running
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/moomerman/zap/control"
)

const usage = `Usage: zap [-socket path] <command> [args]

Commands:
  ls                    list apps and their status
  status <host>         show an app's details
  start <host>          start an app
  stop <host>           stop an app
  restart <host>        restart an app
  logs [-f] <host>      show an app's recent output, -f to follow it
  events [host]         stream status changes and output for all apps, or one
  ngrok <host>          open an ngrok tunnel to an app

Flags:
`

func main() {
	socket := flag.String("socket", control.DefaultSocketPath(), "path to zapd's control socket")
	flag.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), usage)
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	c := control.NewClient(*socket)
	if err := run(ctx, c, flag.Arg(0), flag.Args()[1:]); err != nil {
		if ctx.Err() != nil {
			return
		}
		fmt.Fprintln(os.Stderr, "zap:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, c *control.Client, cmd string, args []string) error {
	switch cmd {
	case "ls", "list":
		apps, err := c.Apps(ctx)
		if err != nil {
			return err
		}
		printApps(os.Stdout, apps, time.Now())
		return nil

	case "status":
		host, err := oneHost(cmd, args)
		if err != nil {
			return err
		}
		app, err := c.App(ctx, host)
		if err != nil {
			return err
		}
		printApp(os.Stdout, app, time.Now())
		return nil

	case "start", "stop", "restart", "ngrok":
		host, err := oneHost(cmd, args)
		if err != nil {
			return err
		}
		actions := map[string]func(context.Context, string) (control.App, error){
			"start":   c.Start,
			"stop":    c.Stop,
			"restart": c.Restart,
			"ngrok":   c.Ngrok,
		}
		app, err := actions[cmd](ctx, host)
		if err != nil {
			return err
		}
		if cmd == "ngrok" {
			fmt.Println(app.Ngrok)
			return nil
		}
		fmt.Printf("%s %s\n", app.Host, app.Status)
		return nil

	case "logs", "log":
		fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
		follow := fs.Bool("f", false, "follow the app's output")
		if err := fs.Parse(args); err != nil {
			return err
		}
		host, err := oneHost(cmd, fs.Args())
		if err != nil {
			return err
		}
		if err := c.Log(ctx, host, os.Stdout); err != nil {
			return err
		}
		if !*follow {
			return nil
		}
		return c.Events(ctx, host, func(e control.Event) error {
			printEvent(os.Stdout, e, false)
			return nil
		})

	case "events":
		var host string
		if len(args) > 0 {
			host = args[0]
		}
		return c.Events(ctx, host, func(e control.Event) error {
			printEvent(os.Stdout, e, true)
			return nil
		})

	default:
		return fmt.Errorf("unknown command %q, run zap -h for help", cmd)
	}
}

func oneHost(cmd string, args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("usage: zap %s <host>", cmd)
	}
	return args[0], nil
}

func printApps(w io.Writer, apps []control.App, now time.Time) {
	if len(apps) == 0 {
		fmt.Fprintln(w, "no apps yet, zapd adds an app when its host is first requested")
		return
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "HOST\tSTATUS\tKIND\tPORT\tPID\tUP\tIDLE")
	for _, a := range apps {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			a.Host, a.Status, a.Kind, dash(a.Port), dash(pid(a.Pid)),
			dash(uptime(a, now)), dash(since(a.LastUsed, now)))
	}
	tw.Flush()
}

func printApp(w io.Writer, a control.App, now time.Time) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	row := func(k, v string) {
		if v != "" {
			fmt.Fprintf(tw, "%s\t%s\n", k, v)
		}
	}
	row("Host", a.Host)
	row("Status", a.Status)
	row("Error", a.Error)
	row("Kind", a.Kind)
	row("Dir", a.Dir)
	row("Command", a.Command)
	row("Proxy", a.Proxy)
	row("Port", a.Port)
	row("Pid", pid(a.Pid))
	row("Up", uptime(a, now))
	row("Idle", since(a.LastUsed, now))
	row("Ngrok", a.Ngrok)
	tw.Flush()
}

func printEvent(w io.Writer, e control.Event, withHost bool) {
	prefix := ""
	if withHost {
		prefix = e.Host + ": "
	}
	switch e.Type {
	case "log":
		fmt.Fprint(w, prefix+e.Line)
		if !strings.HasSuffix(e.Line, "\n") {
			fmt.Fprintln(w)
		}
	default:
		msg := "-> " + e.Status
		if e.Error != "" {
			msg += " (" + e.Error + ")"
		}
		fmt.Fprintf(w, "%s %s[zap] %s\n", e.Time.Format("15:04:05"), prefix, msg)
	}
}

func uptime(a control.App, now time.Time) string {
	if a.Status != "running" {
		return ""
	}
	return since(a.Started, now)
}

func since(t *time.Time, now time.Time) string {
	if t == nil {
		return ""
	}
	return now.Sub(*t).Truncate(time.Second).String()
}

func pid(p int) string {
	if p == 0 {
		return ""
	}
	return fmt.Sprint(p)
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
