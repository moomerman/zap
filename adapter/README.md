# package adapters

This package contains implementations of the Adapter interface that control
different backend applications.

The interface is currently defined as

```go
type Adapter interface {
	Start() error
	Stop(reason error) error
	Status() Status
	Snapshot() Snapshot
	WriteLog(io.Writer)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}
```

Adapters don't manage their own lifecycle: the `zap.Manager` decides when to
start and stop them. An adapter that takes time to boot (the server adapter)
returns from `Start` once its process is launched and reports `running`,
`error` or an unexpected exit through the `StatusFunc` it was created with.

The `ServeHTTP` function means all adapters implement the `http.Handler`
interface.

## Implementations

* `proxy` forwards requests to a URL that is already running
* `server` runs a command on a free port and proxies to it once it is listening
* `static` serves files from a directory

How each one is configured is described in the main README.
