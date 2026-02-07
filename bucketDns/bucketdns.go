package bucketDns

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/containerd/log"
	"github.com/miekg/dns"
	"github.com/redis/go-redis/v9"
	"patchwork.horse/bucketd/config"
)

type DNSHandler struct {
	Redis     *redis.Client
	DnsConfig *config.DnsConfig
}

func StartDnsListener(coreConfig *config.CoreConfig, dnsConfig *config.DnsConfig, redisConfig *config.RedisConfig) {

	h := &DNSHandler{
		Redis: redis.NewClient(&redis.Options{
			Addr:     redisConfig.Address,
			Password: redisConfig.Password,
			DB:       redisConfig.Database,
		}),
		DnsConfig: dnsConfig,
	}

	server := &dns.Server{
		Addr: fmt.Sprintf(":%d", dnsConfig.Port),
		Handler: dns.HandlerFunc(func(w dns.ResponseWriter, m *dns.Msg) {
			h.handleRequest(w, m, dnsConfig)
		}),
		Net: "udp",
	}

	log.L.Infof("Listening on port %d for DNS queries", dnsConfig.Port)
	log.L.Infof("Accepting DNS queries for %s", dnsConfig.FQDN)

	if err := server.ListenAndServe(); err != nil {
		log.L.WithError(err).Fatal("Failed to start DNS server")
	}

}

func (h *DNSHandler) handleRequest(w dns.ResponseWriter, req *dns.Msg, dnsConfig *config.DnsConfig) {
	m := new(dns.Msg)
	m.SetReply(req)

	if len(req.Question) == 0 {
		return
	}

	var err error

	q := &req.Question[0]
	q.Name = strings.ToLower(q.Name)

	if _, err := validateHostname(req, m, dnsConfig); err != nil {
		w.WriteMsg(m) // Returns NXDOMAIN with EDE
		return
	}

	log.L.Infof("Handling %s query for %s", dns.TypeToString[q.Qtype], q.Name)

	switch q.Qtype {
	case dns.TypeTXT:
		err = h.handleTxt(req, m, dnsConfig)
	case dns.TypeA:
		err = handleA(req, m, dnsConfig)
	case dns.TypeAAAA:
		err = handleAAAA(req, m, dnsConfig)
	}

	if err != nil {
		log.L.WithError(err).Error("System error handling DNS request")
		m.SetRcode(req, dns.RcodeServerFailure)
	}

	w.WriteMsg(m)
}

func (h *DNSHandler) handleTxt(req *dns.Msg, res *dns.Msg, dnsConfig *config.DnsConfig) error {

	// Notes:
	// Responses limited to ~255 bytes
	// Assume the leftmost part of the query is our key

	res.SetReply(req)

	key := dns.SplitDomainName(req.Question[0].Name)[0]
	pipe := h.Redis.Pipeline()
	ctx := context.Background()
	cmdGet := pipe.Get(ctx, key)
	cmdTtl := pipe.TTL(ctx, key)

	_, err := pipe.Exec(ctx)

	// Cache miss
	if err == redis.Nil {
		res.SetRcode(req, dns.RcodeNameError)
		return nil
	}

	// Other error, don't handle it here
	if err != nil {
		return err
	}

	val, _ := cmdGet.Result()

	t := &dns.TXT{
		Hdr: dns.RR_Header{
			Name:   req.Question[0].Name,
			Rrtype: dns.TypeTXT,
			Class:  dns.ClassINET,
			Ttl:    uint32(cmdTtl.Val().Seconds()),
		},
		Txt: []string{val},
	}

	res.Answer = append(res.Answer, t)

	return nil
}

func handleA(query *dns.Msg, response *dns.Msg, dnsConfig *config.DnsConfig) error {

	if query.Question[0].Qtype != dns.TypeA {
		return errors.New("handleA expects an A question type")
	}

	a := new(dns.A)
	a.Hdr = dns.RR_Header{
		Name:   query.Question[0].Name,
		Rrtype: dns.TypeA,
		Class:  dns.ClassINET,
		Ttl:    300,
	}

	a.A = dnsConfig.A

	response.Answer = append(response.Answer, a)

	return nil
}

func handleAAAA(query *dns.Msg, response *dns.Msg, dnsConfig *config.DnsConfig) error {
	if query.Question[0].Qtype != dns.TypeAAAA {
		return errors.New("handleAAAA expects an AAAA question type")
	}

	aaaa := new(dns.AAAA)
	aaaa.Hdr = dns.RR_Header{
		Name:   query.Question[0].Name,
		Rrtype: dns.TypeAAAA,
		Class:  dns.ClassINET,
		Ttl:    300,
	}

	aaaa.AAAA = dnsConfig.AAAA

	response.Answer = append(response.Answer, aaaa)

	return nil
}

func validateHostname(query *dns.Msg, response *dns.Msg, dnsConfig *config.DnsConfig) (ok bool, err error) {

	if dns.IsSubDomain(dnsConfig.FQDN, query.Question[0].Name) {
		return true, nil
	}

	response.SetRcode(query, dns.RcodeNameError)
	opt := new(dns.OPT)
	opt.Hdr.Name = "."
	opt.Hdr.Rrtype = dns.TypeOPT
	opt.SetUDPSize(4096)

	ede := new(dns.EDNS0_EDE)
	ede.InfoCode = dns.ExtendedErrorCodeOther
	ede.ExtraText = fmt.Sprintf("Domain must be %s or a subdomain", dnsConfig.FQDN)
	opt.Option = append(opt.Option, ede)

	response.Extra = append(response.Extra, opt)

	return false, fmt.Errorf("Domain must be %s or a subdomain", dnsConfig.FQDN)
}
