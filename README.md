# ⚡ Zap - A development web/proxy server

## About

Zap is a development web/proxy server that knows how to start and manage your
development server processes, and provides SSL access to them.

Zap allows you to specify any command to start a backend server, we've tested it with:

* Elixir/Phoenix
* Ruby/Rails
* Go/Buffalo
* Go/Hugo
* Simple Proxy
* Static HTML

## Features

* SSL - creates a self-signed cert for each domain so you can test SSL in dev
* Process management - start, monitor, spin down idle apps
* Log watching - watches log files and restarts application on certain triggers
* Works on macOS, Linux and Windows (some manual installation required on Linux & Windows)

## Install

Either grab a binary for your platform from the Releases page or grab the code and build your own

```go
go build -o zapd . # build the zapd binary
go build ./cmd/zap # build the zap command
zapd -install # run the installer
```

## Usage

`zap` shows and controls the apps zapd is running. It talks to zapd over a
local unix socket (`~/Library/Application Support/zap/zapd.sock` on macOS,
change it with `zapd -control` and `zap -socket`).

```
zap ls                 # list apps and their status
zap status myapp.test  # show an app's details
zap restart myapp.test # start, stop or restart an app
zap logs -f myapp.test # show and follow an app's output
zap events             # stream status changes and output for every app
```

The same API is available to other clients as JSON over the socket, see
`control/api.go`.

When a request arrives for an app that isn't running, zap starts it and holds
the request until it has booted. If it fails to boot, the response shows the
error and the app's recent output.

## Logs

On macOS zapd logs to `~/Library/Logs/zap/zapd.log`, and each app's output is
written to its own file next to it, named after the host that started it (eg.
`~/Library/Logs/zap/myapp.test.log`). Change the directory with `-logs`, or pass
`-logs=` to send app output to zapd's stdout. If you installed zap before this
change, run `zapd -install` again to move zapd's own log.

## Credits

Inspired by pow (http://pow.cx/) and puma-dev (https://github.com/puma/puma-dev)

## Development

```
go build -o zapd . && pkill zapd # launchd restarts it with the new binary
```
