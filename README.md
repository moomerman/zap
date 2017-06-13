# phx-dev - A development server for phoenix applications

## Features

* Process management - start, monitor, spin down idle apps
* Log watching - watches log files and restarts application on certain triggers
* SSL - creates a self-signed cert for each domain so you can test SSL in dev

## Usage

Add an application

`ln -sf /path/to/phoenix/app ~/.phx-dev`

Configure (optional)

```
PORT=4001
CMD=mix phx.server
```

Hosts

```
127.0.0.1 yourapp.phx
```

## Wishlist

* Linux, Windows Support
* Status UI

## Credits

Inspired by pow (http://pow.cx/) and puma-dev (https://github.com/puma/puma-dev)
