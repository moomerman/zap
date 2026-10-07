package dns

import (
	"net"
	"testing"

	"github.com/miekg/dns"
)

func TestResponderAnswersLoopback(t *testing.T) {
	r := &Responder{Address: "127.0.0.1:0", Domains: []string{"test"}}
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &dns.Server{PacketConn: pc, Handler: dns.HandlerFunc(r.handleDNS)}
	go srv.ActivateAndServe()
	defer srv.Shutdown()

	for qtype, want := range map[uint16]string{dns.TypeA: "127.0.0.1", dns.TypeAAAA: "::1"} {
		m := new(dns.Msg)
		m.SetQuestion("phx.test.", qtype)
		in, err := dns.Exchange(m, pc.LocalAddr().String())
		if err != nil {
			t.Fatal(err)
		}
		if !in.Authoritative || len(in.Answer) != 1 || in.Answer[0].Header().Rrtype != qtype {
			t.Fatalf("%s: got %v", dns.TypeToString[qtype], in)
		}
		var got net.IP
		switch rr := in.Answer[0].(type) {
		case *dns.A:
			got = rr.A
		case *dns.AAAA:
			got = rr.AAAA
		}
		if got.String() != want {
			t.Errorf("%s: got %s, want %s", dns.TypeToString[qtype], got, want)
		}
	}
}
