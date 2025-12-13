package objectDns

import (
	"context"
	"errors"
	"fmt"

	"github.com/miekg/dns"
	"github.com/redis/go-redis/v9"
)

// Not great to define Redis options everywhere they're used
var redisOptions = &redis.Options{
	Addr:     "redis:6379",
	Password: "",
	DB:       0,
	Protocol: 2,
}

func StartDnsListener() {

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

		switch q.Qtype {
		case dns.TypeTXT:
			handleTxt(r, m)
		}
	}

	w.WriteMsg(m)
}

func handleTxt(query *dns.Msg, response *dns.Msg) error {

	if query.Question[0].Qtype != dns.TypeTXT {
		return errors.New("handleTxt expects a TXT question type")
	}

	rdb, ctx, parts :=
		redis.NewClient(redisOptions),
		context.Background(),
		dns.SplitDomainName(query.Question[0].Name)

	key := parts[0]

	fmt.Printf("Attempting cache hit for DNS query part %s...\n", key)

	pipe := rdb.Pipeline()
	getCmd := pipe.Get(ctx, key)
	ttlCmd := pipe.TTL(ctx, key)

	_, err := pipe.Exec(ctx)

	if err != nil && err != redis.Nil {
		response.SetRcode(query, dns.RcodeServerFailure)
		return err
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
	return nil
}
