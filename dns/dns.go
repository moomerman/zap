package dns

import (
	"log"
	"net"
	"sync"

	"github.com/miekg/dns"
)

// DefaultAddress is the default address for the DNS server
const DefaultAddress = ":9253"

// Responder holds the configuration for the DNS server
type Responder struct {
	Address string
	Domains []string

	udpServer *dns.Server
	tcpServer *dns.Server
}

// Serve starts the DNS server
func (d *Responder) Serve() {
	for _, domain := range d.Domains {
		dns.HandleFunc(domain+".", d.handleDNS)
	}

	addr := d.Address
	if addr == "" {
		addr = DefaultAddress
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		d.udpServer = &dns.Server{Addr: addr, Net: "udp", TsigSecret: nil}
		if err := d.udpServer.ListenAndServe(); err != nil {
			log.Printf("[dns] udb server stopped unexpectedly %v\n", err)
		}
	}()

	go func() {
		defer wg.Done()
		d.tcpServer = &dns.Server{Addr: addr, Net: "tcp", TsigSecret: nil}
		if err := d.tcpServer.ListenAndServe(); err != nil {
			log.Printf("[dns] tcp server stopped unexpectedly %v\n", err)
		}
	}()

	log.Println("[dns]", "listening at udp/tcp", addr)
	wg.Wait()
}

// Stop stops the DNS servers
func (d *Responder) Stop() {
	if err := d.udpServer.Shutdown(); err != nil {
		log.Println("[dns] error shutting down UDP server", err)
	}
	if err := d.tcpServer.Shutdown(); err != nil {
		log.Println("[dns] error shutting down TCP server", err)
	}
}

func (d *Responder) handleDNS(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	if len(r.Question) > 0 {
		q := r.Question[0]
		hdr := dns.RR_Header{Name: q.Name, Class: dns.ClassINET, Ttl: 0}
		switch q.Qtype {
		case dns.TypeA:
			hdr.Rrtype = dns.TypeA
			m.Answer = append(m.Answer, &dns.A{Hdr: hdr, A: net.IPv4(127, 0, 0, 1)})
		case dns.TypeAAAA:
			hdr.Rrtype = dns.TypeAAAA
			m.Answer = append(m.Answer, &dns.AAAA{Hdr: hdr, AAAA: net.IPv6loopback})
		}
	}

	w.WriteMsg(m)
}
