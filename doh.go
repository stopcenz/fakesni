package main

import (
    "context"
    "errors"
    "time"
    "github.com/likexian/doh-go"
    "github.com/likexian/doh-go/dns"
)

func getIp(hostname string) (ip string, err error) {
	// init a context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// init doh client, auto select the fastest provider base on your like
	// you can also use as: c := doh.Use(), it will select from all providers
	c := doh.Use(doh.CloudflareProvider, doh.GoogleProvider)

	// do doh query
	rsp, err := c.Query(ctx, dns.Domain(hostname), dns.TypeA)
	if err != nil {
	    return "", err
	}
	c.Close()
	
	for _, a := range rsp.Answer {
	    if len(a.Data) > 0 {
	      return a.Data, nil
	    }
	}
	return "", errors.New("DoH: empty response")
}
