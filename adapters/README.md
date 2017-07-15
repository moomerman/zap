# package adapters

This package contains implementations of the Adapter interface that control
different backend applications.

The interface is currently defined as

```go
type Adapter interface {
	Command() *exec.Cmd
	Start() error
	Stop() error
	WriteLog(io.Writer)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}
```

The `ServeHTTP` function means all adapters implement the `http.Handler`
interface.

## Working implementations

### Simple Proxy

A proxy is configured by creating a file in the `~/.phx-dev` folder containing the
host/port combination that you want to proxy to.  You can specify just a port and
it will assume localhost.

`echo "http://127.0.0.1:3000" > ~/.phx-dev/mysite.dev`
`echo "4000" > ~/.phx-dev/othersite.dev`

### Elixir/Phoenix

When a `mix.exs` file is detected in the root of the dir that a symlink points to
and contains the `phoenix` package then a `mix phx.server` server will be launched
on a random port and requests will be proxied to that application.

To configure a phoenix application simply create a symbolic link to the
phoenix application and update your `config/dev.exs` file to allow phx-dev
to override the default 4000 http port.

`ln -sf /path/to/phoenix/app ~/.phx-dev/mysite.dev`

`config/mix.exs`
```elixir
config :your_app, YourApp.Web.Endpoint,
  http: [port: System.get_env("PHX_PORT") || 4000],
```

### Static HTML

To enable a static HTML site, simply symlink to a the public directory
where the static files live.  Files in the directory will be served, if a directory
root is requested `index.html` files will be served if they exist.

`ln -sf /path/to/static/public ~/.phx-dev/mysite.dev`

## TODO

* Ruby/Rails
* Ruby/Hanami
* Ruby/Rack ?
* Go/Buffalo
* Node ?
* PHP ?
