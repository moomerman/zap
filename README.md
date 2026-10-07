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

```
make            # builds bin/zapd and bin/zap
bin/zapd -install # run the installer
```

The installer:

* writes `/etc/resolver/test` so `*.test` resolves to zap's DNS responder on
  `127.0.0.1:9253` (it uses sudo for this, so run it as yourself and enter
  your password when asked). Pick other domains with `-domains=test:localhost`
  on both `-install` and `-uninstall`.
* creates the certificate authority zap signs each host's certificate with.
* installs a launch agent that runs the zapd binary it was run from, listening
  directly on `127.0.0.1:80` and `127.0.0.1:443`. Run it again if you move the
  binary, or to change the `-http`, `-https`, `-dns` or `-domains` it passes.

If another file in `/etc/resolver` also claims `.test` (Apple's `container`
tool writes `containerization.test`), the installer warns about it. zap's file
asks to be tried first, but if `.test` hosts don't resolve, check
`scutil --dns` and remove the other file.

`bin/zapd -uninstall` removes the launch agent and the resolver files zap wrote.

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
make && pkill zapd # launchd restarts it with the new binary
```
