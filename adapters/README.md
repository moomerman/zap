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

## Current implementations

### Simple Proxy

A proxy is configured by creating a file in the `~/.phx-dev` folder with the
host/port combination that you want to proxy to.

eg. `echo "http://127.0.0.1:3000" > ~/.phx-dev/mysite.dev`

### Elixir/Phoenix

If a `mix.exs` file is detected in the root of the dir that the symlink points to
and contains the `phoenix` package then a phoenix server will be launched and
requests will be proxied to that port.

eg. `ln -sf /path/to/phoenix/app ~/.phx-dev/mysite.dev`

### Static HTML

To enable a static HTML site, simply simlink to a the public directory
where the static files live.

eg. `ln -sf /path/to/static/public ~/.phx-dev/mysite.dev`

## TODO

* Ruby/Rails
* Ruby/Hanami
* Ruby/Rack ?
* Go/Buffalo
* Node ?
* PHP ?
