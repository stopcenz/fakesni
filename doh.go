package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"time"
	"github.com/likexian/doh-go/dns"
	tris "github.com/stopcenz/tls-tris"
	//tris "tris"
)

const typeA = 1
const typeCNAME = 5
const typeTXT = 16

func dTime(start int64, hostname string) {
	if DBG {
		dt := (time.Now().UnixNano() - start) / 1e6
		if dt > 0 {
			log.Print("doh: ", dt, "ms for ", hostname)
		}
	}
}

func seekESNIKeys(hostname string) (*tris.ESNIKeys, error) {
	start := time.Now().UnixNano()
	ctx, cancel := context.WithTimeout(context.Background(), 20 * time.Second)
	defer cancel()
	rsp, err := dohClient.Query(
		ctx,
		dns.Domain("_esni." + hostname),
		dns.TypeTXT,
	)
	dTime(start, hostname)
	if err != nil {
		return nil, errors.New("ESNI keys: " + err.Error())
	}
	for _, a := range rsp.Answer {
		if a.Type != typeTXT {
			continue
		}
		// Quad9 response is a quoted string
		data := strings.Trim(a.Data, " \t\"'")
		rawEsniKeys, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			log.Print("Error: DoH: ", err.Error())
			continue
		}
		return tris.ParseESNIKeys(rawEsniKeys)
	}
	return nil, errors.New("ESNI keys not found for '" + hostname + "'")
}

func isEquiv(a []string, b []string) bool {
	if a == nil || b == nil || len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func resolveAsync(alias *Alias, ch chan error) {
	start := time.Now().UnixNano()
	noError := errors.New("")
	hostname := alias.Hostname
	ctx, cancel := context.WithTimeout(context.Background(), 30 * time.Second)
	defer cancel()
	rsp, err := dohClient.Query(ctx, dns.Domain(hostname), dns.TypeA)
	dTime(start, hostname)
	if err != nil {
		if ch != nil {
			ch <- err
		}
		return
	}
	names := []string{ hostname }
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
		if ch != nil {
			ch <- errors.New("No match was found for '" + hostname + "'")
		}
		return
	}
	sort.Strings(ips)
	if isEquiv(alias.IPs[:], ips[:]) {
		if ch != nil {
			ch <- noError
		}
		return
	}
	alias.IPs = ips
	if ch != nil {
		ch <- noError
	}
	var s string
	if len(ips) > 1 {
		s = "['" + strings.Join(ips, "', '") + "']"
	} else {
		s = "'" + ips[0] + "'"
	}
	dt := (time.Now().UnixNano() - start) / 1e6
	s += fmt.Sprintf(", %dms.", dt)
	log.Print("Resolved '", alias.Hostname, "' to ", s)
}


func resolve(c *Context) error {
	if c.Alias.IPs != nil && len(c.Alias.IPs) > 0 {
		go resolveAsync(c.Alias, nil)
		c.IP = c.Alias.IPs[rand.Intn(len(c.Alias.IPs))]
		return nil
	}
	ch := make(chan error)
	go resolveAsync(c.Alias, ch)
	err := <- ch
	if err.Error() != "" {
		return err
	}
	c.IP = c.Alias.IPs[rand.Intn(len(c.Alias.IPs))]
	return nil
}
