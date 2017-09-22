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

A proxy is configured by creating a file in the `~/.zap` folder containing the
host/port combination that you want to proxy to.  You can specify just a port and
it will assume localhost.

`echo "http://127.0.0.1:3000" > ~/.zap/mysite.dev`
`echo "4000" > ~/.zap/othersite.dev`

### Elixir/Phoenix

When a `mix.exs` file is detected in the root of the dir that a symlink points to
and contains the `phoenix` package then a `mix phx.server` server will be launched
on a random port and requests will be proxied to that application.

To configure a phoenix application simply create a symbolic link to the
phoenix application and update your `config/dev.exs` file to allow zap
to override the default 4000 http port.

`ln -sf /path/to/phoenix/app ~/.zap/mysite.dev`

`config/mix.exs`
```elixir
config :your_app, YourApp.Web.Endpoint,
  http: [port: System.get_env("PHX_PORT") || 4000],
```

### Ruby/Rails

If a `Gemfile` is detected in the root of the project then the Ruby/Rails
adapter is used.

### Go/Buffalo

If a `.buffalo.dev.yml` file is found then a Go/Buffalo developent server
backend is started.

### Go/Hugo

If a `config.toml` file is found in the root of the project then a Go/Hugo
development backend is started.

### Static HTML

To enable a static HTML site, simply symlink to a the public directory
where the static files live.  Files in the directory will be served, if a directory
root is requested `index.html` files will be served if they exist.

`ln -sf /path/to/static/public ~/.zap/mysite.dev`

## TODO

* Ruby/Hanami
* Ruby/Rack
* Ruby/Rakefile
* Docker (Compose)
* Procfile
* Node/package.json
* PHP
