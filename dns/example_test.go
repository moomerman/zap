package dns_test

import (
	"log"

	"github.com/moomerman/zap/dns"
)

func Example() {
	responder := dns.Responder{
		Address: "127.0.0.1:9253",
		Domains: []string{"test"},
	}
	log.Println("* DNSServer", responder.Address)

	responder.Serve()
}
