package dns_test

import (
	"fmt"
	"log"

	"github.com/moomerman/zap/dns"
)

func Example() {

	port := 9253
	domains := []string{"test", "dev"}

	if err := dns.ConfigureResolver(domains, port, "example"); err != nil {
		log.Println(err)
		panic("couldn't configure resolver")
	}

	var dns dns.Responder
	dns.Address = fmt.Sprintf("127.0.0.1:%d", port)
	log.Println("* DNSServer", dns.Address)

	dns.Serve()
}
