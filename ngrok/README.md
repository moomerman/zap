# package ngrok

This package provides a wrapper around the ngrok executable so you
can programatically start and stop tunnels for a given host/port.

## Usage example

```go
ngrok, err := ngrok.StartTunnel("example.dev", 80)
if err != nil {
	log.Fatal("unable to start tunnel", err)
}

fmt.Println(ngrok.URL) // => https://a1b2c3d4.ngrok.io

// use the tunnel until you've finished with it

if err := ngrok.Stop(); err != nil {
	log.Println("error stopping tunnel", err)
}
```
