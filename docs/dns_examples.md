# miekg/dns Usage Examples

This document provides simple examples of how to use the `github.com/miekg/dns` package in Go.

## Prerequisites

Ensure you have the package installed:

```bash
go get github.com/miekg/dns
```

## Example 1: Simple DNS Server (TXT Record)

This example demonstrates how to set up a simple DNS server that responds to TXT queries.

```go
package main

import (
	"log"
`	"github.com/miekg/dns"`
)

func handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	// Check if there are questions
	if len(r.Question) > 0 {
		q := r.Question[0]
		
		switch q.Qtype {
		case dns.TypeTXT:
			t := new(dns.TXT)
			t.Hdr = dns.RR_Header{
				Name:   q.Name,
				Rrtype: dns.TypeTXT,
				Class:  dns.ClassINET,
				Ttl:    0,
			}
			t.Txt = []string{"Hello from miekg/dns!"}
			m.Answer = append(m.Answer, t)
		}
	}

	w.WriteMsg(m)
}

func main() {
	// Attach the handler function
	dns.HandleFunc(".", handleRequest)

	// Start the server on port 8053
	server := &dns.Server{Addr: ":8053", Net: "udp"}
	log.Printf("Starting DNS server on port 8053...")
	
	err := server.ListenAndServe()
	defer server.Shutdown()
	
	if err != nil {
		log.Fatalf("Failed to start server: %s\n ", err.Error())
	}
}
```

## Example 2: Parsing Queries

This example shows how to inspect the incoming query in the handler.

```go
package main

import (
	"fmt"
	"github.com/miekg/dns"
)

func handleQueryParsing(w dns.ResponseWriter, r *dns.Msg) {
	for _, q := range r.Question {
		fmt.Printf("Query Name: %s\n", q.Name)
		fmt.Printf("Query Type: %d\n", q.Qtype)
		fmt.Printf("Query Class: %d\n", q.Qclass)
		
		// Check for specific types
		switch q.Qtype {
		case dns.TypeA:
			fmt.Println("Received A record query")
		case dns.TypeAAAA:
			fmt.Println("Received AAAA record query")
		case dns.TypeTXT:
			fmt.Println("Received TXT record query")
		}
	}
	
	// Send an empty reply or handle accordingly
	m := new(dns.Msg)
	m.SetReply(r)
	w.WriteMsg(m)
}
```
