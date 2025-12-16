package objectDns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/miekg/dns"
	"github.com/redis/go-redis/v9"
)

// Patchwork todo:
// - Centralize error handling, don't generate EDNS0_EDEs in every handler
// - Better logging
// - Read A and AAAA values from environment variables

var redOpts *redis.Options

func StartDnsListener(redisOptions *redis.Options) {

	redOpts = redisOptions

	dns.HandleFunc(".", handleRequest)

	server := &dns.Server{Addr: ":8053", Net: "udp"}

	println("Starting DNS server on port 8053...")

	err := server.ListenAndServe()
	defer server.Shutdown()

	if err != nil {
		println("Failed to start server: %s\n ", err.Error())
	}
}

func handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	// Check if there are questions
	if len(r.Question) > 0 {
		q := r.Question[0]

		q.Name = strings.ToLower(q.Name)

		switch q.Qtype {
		case dns.TypeTXT:
			handleTxt(r, m)
		case dns.TypeA:
			fmt.Printf("Handling A request for %s...\n", q.Name)
			handleA(r, m)
		case dns.TypeAAAA:
			fmt.Printf("Handling AAAA request for %s...\n", q.Name)
			handleAAAA(r, m)
		}

		w.WriteMsg(m)
	}
}

func handleTxt(query *dns.Msg, response *dns.Msg) error {

	if query.Question[0].Qtype != dns.TypeTXT {
		return errors.New("handleTxt expects a TXT question type")
	}

	rdb, ctx, parts :=
		redis.NewClient(redOpts),
		context.Background(),
		dns.SplitDomainName(query.Question[0].Name)

	key := parts[0]

	fmt.Printf("Attempting to satisfy DNS object request for key %s...\n", key)

	pipe := rdb.Pipeline()
	getCmd := pipe.Get(ctx, key)
	ttlCmd := pipe.TTL(ctx, key)

	_, err := pipe.Exec(ctx)

	// Retrieval error
	if err != nil && err != redis.Nil {
		fmt.Printf("Error resolving value for key %s...\n", key)
		response.SetRcode(query, dns.RcodeServerFailure)
		return err
	}

	// Cache miss
	if err != nil && err == redis.Nil {
		fmt.Printf("Cache miss for DNS query part %s...\n", key)
		response.SetRcode(query, dns.RcodeNameError)
		return errors.New("cache miss")
	}

	value, _ := getCmd.Result()
	ttl, _ := ttlCmd.Result()

	t := new(dns.TXT)
	t.Hdr = dns.RR_Header{
		Name:   query.Question[0].Name,
		Rrtype: dns.TypeTXT,
		Class:  dns.ClassINET,
		Ttl:    uint32(ttl.Seconds()),
	}

	t.Txt = []string{value}

	response.Answer = append(response.Answer, t)

	fmt.Printf("Cache hit! Key TTL is: %s...\n", ttl)

	return nil
}

func handleA(query *dns.Msg, response *dns.Msg) error {
	if query.Question[0].Qtype != dns.TypeA {
		return errors.New("handleA expects an A question type")
	}

	// Question section must be for *.object.patchwork.horse or object.patchwork.horse
	if !dns.IsSubDomain("object.patchwork.horse.", query.Question[0].Name) && query.Question[0].Name != "object.patchwork.horse." {
		response.SetRcode(query, dns.RcodeNameError)

		// Add extended DNS error information
		opt := new(dns.OPT)
		opt.Hdr.Name = "."
		opt.Hdr.Rrtype = dns.TypeOPT
		opt.SetUDPSize(4096)

		ede := new(dns.EDNS0_EDE)
		ede.InfoCode = dns.ExtendedErrorCodeOther
		ede.ExtraText = "Domain must be object.patchwork.horse or a subdomain"
		opt.Option = append(opt.Option, ede)

		response.Extra = append(response.Extra, opt)

		return errors.New("invalid domain; must be object.patchwork.horse or a subdomain")
	}

	// Huge hack, forgive me. Return hard coded A values
	a := new(dns.A)
	a.Hdr = dns.RR_Header{
		Name:   query.Question[0].Name,
		Rrtype: dns.TypeA,
		Class:  dns.ClassINET,
		Ttl:    300,
	}
	a.A = net.ParseIP("50.116.57.102")

	response.Answer = append(response.Answer, a)

	return nil
}

func handleAAAA(query *dns.Msg, response *dns.Msg) error {
	if query.Question[0].Qtype != dns.TypeAAAA {
		return errors.New("handleAAAA expects an AAAA question type")
	}

	// Question section must be for *.object.patchwork.horse or object.patchwork.horse
	if !dns.IsSubDomain("object.patchwork.horse.", query.Question[0].Name) && query.Question[0].Name != "object.patchwork.horse." {
		response.SetRcode(query, dns.RcodeNameError)

		// Add extended DNS error information
		opt := new(dns.OPT)
		opt.Hdr.Name = "."
		opt.Hdr.Rrtype = dns.TypeOPT
		opt.SetUDPSize(4096)

		ede := new(dns.EDNS0_EDE)
		ede.InfoCode = dns.ExtendedErrorCodeOther
		ede.ExtraText = "Domain must be object.patchwork.horse or a subdomain"
		opt.Option = append(opt.Option, ede)

		response.Extra = append(response.Extra, opt)

		return errors.New("invalid domain; must be object.patchwork.horse or a subdomain")
	}

	// Huge hack, Epona forgive me. Return hard coded AAAA values
	aaaa := new(dns.AAAA)
	aaaa.Hdr = dns.RR_Header{
		Name:   query.Question[0].Name,
		Rrtype: dns.TypeAAAA,
		Class:  dns.ClassINET,
		Ttl:    300,
	}
	aaaa.AAAA = net.ParseIP("2600:3c03::f03c:95ff:fe5d:294f")

	response.Answer = append(response.Answer, aaaa)

	return nil
}
