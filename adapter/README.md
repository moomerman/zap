# package adapter

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

The `ServeHTTP` function means all adapters also implement the `http.Handler`
interface.

## Current implementations

* Static HTML
* Simple Proxy
* Elixir Phoenix

## TODO

* Ruby Rails
* Node ?
* PHP ?
* Go Buffalo
