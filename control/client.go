package control

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// DefaultSocketPath is where zapd serves the control API:
// ~/Library/Application Support/zap/zapd.sock on macOS and
// ~/.config/zap/zapd.sock on Linux
func DefaultSocketPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "zapd.sock")
	}
	return filepath.Join(dir, "zap", "zapd.sock")
}

// Client talks to zapd's control API
type Client struct {
	http *http.Client
}

// NewClient returns a client for the control API on the unix socket at path
func NewClient(path string) *Client {
	var d net.Dialer
	return &Client{http: &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return d.DialContext(ctx, "unix", path)
			},
		},
	}}
}

// Apps returns every app zapd knows about
func (c *Client) Apps(ctx context.Context) ([]App, error) {
	var apps []App
	return apps, c.do(ctx, http.MethodGet, "/v1/apps", &apps)
}

// App returns the app serving host
func (c *Client) App(ctx context.Context, host string) (App, error) {
	var app App
	return app, c.do(ctx, http.MethodGet, appPath(host, ""), &app)
}

// Start starts the app serving host
func (c *Client) Start(ctx context.Context, host string) (App, error) {
	return c.action(ctx, host, "start")
}

// Stop stops the app serving host
func (c *Client) Stop(ctx context.Context, host string) (App, error) {
	return c.action(ctx, host, "stop")
}

// Restart restarts the app serving host
func (c *Client) Restart(ctx context.Context, host string) (App, error) {
	return c.action(ctx, host, "restart")
}

// Ngrok opens an ngrok tunnel to the app serving host
func (c *Client) Ngrok(ctx context.Context, host string) (App, error) {
	return c.action(ctx, host, "ngrok")
}

func (c *Client) action(ctx context.Context, host, action string) (App, error) {
	var app App
	return app, c.do(ctx, http.MethodPost, appPath(host, action), &app)
}

// Log writes the recent output of the app serving host to w
func (c *Client) Log(ctx context.Context, host string, w io.Writer) error {
	res, err := c.request(ctx, http.MethodGet, appPath(host, "log"))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, err = io.Copy(w, res.Body)
	return err
}

// Events calls f with each event until ctx is done, f returns an error or
// zapd closes the stream. With a host, only events for that app are sent.
func (c *Client) Events(ctx context.Context, host string, f func(Event) error) error {
	path := "/v1/events"
	if host != "" {
		path += "?host=" + url.QueryEscape(host)
	}
	res, err := c.request(ctx, http.MethodGet, path)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	scanner := bufio.NewScanner(res.Body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		data, ok := strings.CutPrefix(scanner.Text(), "data: ")
		if !ok {
			continue
		}
		var e Event
		if err := json.Unmarshal([]byte(data), &e); err != nil {
			return err
		}
		if err := f(e); err != nil {
			return err
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return scanner.Err()
}

func (c *Client) do(ctx context.Context, method, path string, v any) error {
	res, err := c.request(ctx, method, path)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return json.NewDecoder(res.Body).Decode(v)
}

// request sends a request and turns error responses into errors
func (c *Client) request(ctx context.Context, method, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, "http://zapd"+path, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("can't reach zapd, is it running? %w", err)
	}
	if res.StatusCode >= 300 {
		defer res.Body.Close()
		var e errorResponse
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil || e.Error == "" {
			return nil, errors.New(res.Status)
		}
		return nil, errors.New(e.Error)
	}
	return res, nil
}

func appPath(host, action string) string {
	path := "/v1/apps/" + url.PathEscape(host)
	if action != "" {
		path += "/" + action
	}
	return path
}
