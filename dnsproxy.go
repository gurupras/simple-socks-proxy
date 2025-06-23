package simplesocksproxy

import (
	"context"
	"log"
	"net"

	"github.com/miekg/dns"
)

type DNSProxy struct {
	Addr string // e.g. ":5353"
}

func (d *DNSProxy) Start() error {
	server := &dns.Server{Addr: d.Addr, Net: "udp"}
	dns.HandleFunc(".", d.handleRequest)
	log.Printf("Starting DNS proxy on %s\n", d.Addr)
	return server.ListenAndServe()
}

func (d *DNSProxy) handleRequest(w dns.ResponseWriter, req *dns.Msg) {
	resp := new(dns.Msg)
	resp.SetReply(req)

	resolver := &net.Resolver{}

	for _, q := range req.Question {
		switch q.Qtype {
		case dns.TypeA:
			ips, err := resolver.LookupHost(context.Background(), q.Name)
			if err != nil {
				log.Printf("Lookup failed for %s: %v", q.Name, err)
				continue
			}
			for _, ip := range ips {
				if parsed := net.ParseIP(ip); parsed != nil && parsed.To4() != nil {
					rr := &dns.A{
						Hdr: dns.RR_Header{
							Name:   q.Name,
							Rrtype: dns.TypeA,
							Class:  dns.ClassINET,
							Ttl:    300,
						},
						A: parsed.To4(),
					}
					resp.Answer = append(resp.Answer, rr)
				}
			}
		case dns.TypeAAAA:
			ips, err := resolver.LookupHost(context.Background(), q.Name)
			if err != nil {
				log.Printf("Lookup failed for %s: %v", q.Name, err)
				continue
			}
			for _, ip := range ips {
				if parsed := net.ParseIP(ip); parsed != nil && parsed.To16() != nil && parsed.To4() == nil {
					rr := &dns.AAAA{
						Hdr: dns.RR_Header{
							Name:   q.Name,
							Rrtype: dns.TypeAAAA,
							Class:  dns.ClassINET,
							Ttl:    300,
						},
						AAAA: parsed.To16(),
					}
					resp.Answer = append(resp.Answer, rr)
				}
			}
		default:
			log.Printf("Unsupported query type: %d", q.Qtype)
		}
	}

	_ = w.WriteMsg(resp)
}
