# ⚡ Zap - A development web/proxy server

## About

Zap is a development web/proxy server that knows how to start and manage your
development server processes, and provides SSL access to them.

Zap knows how to manage a number of `Backends` including:

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
go build -o zapd main.go # build the zapd binary
zapd -install # run the installer
```

## Wishlist

* Status UI

## Credits

Inspired by pow (http://pow.cx/) and puma-dev (https://github.com/puma/puma-dev)

## Development

To recompile the HTML templates, build and restart the server

```
pushd zap; go-bindata -pkg zap -o templates.go templates/; popd && go build -o zapd main.go && pkill zapd
```
