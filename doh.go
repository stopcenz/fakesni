package main

import (
	"context"
	"encoding/base64"
	"errors"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"time"
	"github.com/likexian/doh-go"
	"github.com/likexian/doh-go/dns"
	tris "github.com/stopcenz/tls-tris"
)

const typeA = 1
const typeCNAME = 5
const typeTXT = 16

func getIp(hostname string) (ip string, esniKeys *tris.ESNIKeys, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// init doh client, auto select the fastest provider base on your like
	// you can also use as: c := doh.Use(), it will select from all providers
	c := doh.Use(doh.CloudflareProvider, doh.GoogleProvider, doh.Quad9Provider)

	// do doh query
	rsp, err := c.Query(ctx, dns.Domain("_esni." + hostname), dns.TypeTXT)
	if err != nil {
		log.Print("Get ESNI keys of ", hostname, ": ", err.Error())
	} else {
		for _, a := range rsp.Answer {
			if a.Type != typeTXT {
				continue
			}
			// Quad9Provider response is a quoted string
			txt := strings.Trim(a.Data, " \t\"'")
			rawEsniKeys, err := base64.StdEncoding.DecodeString(txt)
		
			if err != nil {
				log.Print("DecodeString: ", err.Error())
			} else {
				esniKeys, err = tris.ParseESNIKeys(rawEsniKeys)
				if err == nil {
					break
				} else {
					esniKeys = nil
					log.Print("ParseESNIKeys: ", err.Error())
				}
			}
		}
	}
	rsp, err = c.Query(ctx, dns.Domain(hostname), dns.TypeA)
	if err != nil {
		return "", nil, err
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
		return "", nil, errors.New("DoH: No match was found for '" + hostname + "'")
	}
	return ips[rand.Intn(len(ips))], esniKeys, nil
}
