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
URL that you want to proxy to.  

`echo "proxy: http://127.0.0.1:3000" > ~/.zap/mysite.test`

### Elixir/Phoenix

Update your `config/dev.exs` file to allow zap to override the default 4000 http port.

`config/mix.exs`
```elixir
config :your_app, YourApp.Web.Endpoint,
  http: [port: System.get_env("PHX_PORT") || 4000],
```

~/.zap/phoenixapp.test

```
dir: /path/to/phoenix/app
command: mix phx.server
port: PHX_PORT
```

### Ruby/Rails

~/.zap/railsapp.test

```
dir: /path/to/rails/app
command: bin/rails s -p %s
```

### Go/Buffalo

~/.zap/buffaloapp.test

```
dir: /path/to/buffalo/app
command: buffalo dev
```

### Go/Hugo

~/.zap/buffaloapp.test

```
dir: /path/to/hugo/app
cmd: hugo server -D -p %s -b https://%s/ --appendPort=false --liveReloadPort=443 --navigateToChanged
```

### Static HTML

To enable a static HTML site, simply specify the public directory
where the static files live.  Files in the directory will be served, if a directory
root is requested `index.html` files will be served if they exist.

~/.zap/staticapp.test

```
dif: /path/to/static/app
```
