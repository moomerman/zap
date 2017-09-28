package ngrok

import (
	"fmt"
	"log"
	"testing"
)

func TestNgrok(t *testing.T) {

	ngrok, err := StartTunnel("example.dev", 80)
	if err != nil {
		log.Fatal("unable to start tunnel", err)
	}

	fmt.Println(ngrok.URL)

	if err := ngrok.Stop(); err != nil {
		log.Println("unable to stop tunnel", err)
	}
}
