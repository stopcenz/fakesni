package main

import (
	"context"
	"errors"
	"math/rand"
	"regexp"
	"strings"
	"time"
	"github.com/likexian/doh-go"
	"github.com/likexian/doh-go/dns"
)

const typeA = 1
const typeCNAME = 5

func getIp(hostname string) (ip string, err error) {
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
	names := []string{hostname}
	for _, a := range rsp.Answer {
		if a.Type == typeCNAME {
			names = append(names, strings.Trim(a.Data, "."))
		}
	}
	ips := []string{}
	ipRe := regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
	for _, a := range rsp.Answer[:] {
		if a.Type == typeA && ipRe.MatchString(a.Data){
			for _, name := range names {
				if name == strings.Trim(a.Name, ".") {
					ips = append(ips, a.Data)
				}
			}
		}
	}
	if len(ips) < 1 {
		return "", errors.New("DoH: No match was found for '" + hostname + "'")
	}
	return ips[rand.Intn(len(ips))], nil
}
