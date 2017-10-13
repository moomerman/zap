package dns

import (
	"log"
	"net"
	"time"

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

	go func() {
		d.udpServer = &dns.Server{Addr: addr, Net: "udp", TsigSecret: nil}
		err := d.udpServer.ListenAndServe()
		log.Println("[dns]", "udp server stopped", err)
	}()

	go func() {
		d.tcpServer = &dns.Server{Addr: addr, Net: "tcp", TsigSecret: nil}
		err := d.tcpServer.ListenAndServe()
		log.Println("[dns]", "tcp server stopped", err)
	}()

	log.Println("[dns]", "listening at udp/tcp", addr)
}

// Stop stops the DNS servers
func (d *Responder) Stop() {
	if err := d.udpServer.Shutdown(); err != nil {
		log.Println("[dns] error shutting down UDP server")
	}
	if err := d.tcpServer.Shutdown(); err != nil {
		log.Println("[dns] error shutting down TCP server")
	}
}

func (d *Responder) handleDNS(w dns.ResponseWriter, r *dns.Msg) {
	var (
		v4 bool
		rr dns.RR
		a  net.IP
	)

	dom := r.Question[0].Name

	m := new(dns.Msg)
	m.SetReply(r)
	if ip, ok := w.RemoteAddr().(*net.UDPAddr); ok {
		a = ip.IP
		v4 = a.To4() != nil
	}
	if ip, ok := w.RemoteAddr().(*net.TCPAddr); ok {
		a = ip.IP
		v4 = a.To4() != nil
	}

	if v4 {
		rr = new(dns.A)
		rr.(*dns.A).Hdr = dns.RR_Header{Name: dom, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0}
		rr.(*dns.A).A = a.To4()
	} else {
		rr = new(dns.AAAA)
		rr.(*dns.AAAA).Hdr = dns.RR_Header{Name: dom, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 0}
		rr.(*dns.AAAA).AAAA = a
	}

	switch r.Question[0].Qtype {
	case dns.TypeAAAA, dns.TypeA:
		m.Answer = append(m.Answer, rr)
	}

	if r.IsTsig() != nil {
		if w.TsigStatus() == nil {
			m.SetTsig(r.Extra[len(r.Extra)-1].(*dns.TSIG).Hdr.Name, dns.HmacMD5, 300, time.Now().Unix())
		}
	}

	w.WriteMsg(m)
}
