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
	//"github.com/likexian/doh-go"
	"github.com/likexian/doh-go/dns"
	tris "github.com/stopcenz/tls-tris"
	//tris "tris"
)

const typeA = 1
const typeCNAME = 5
const typeTXT = 16


func seekEsniKeys(hostname string) *tris.ESNIKeys {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	rsp, err := dohClient.Query(
		ctx,
		dns.Domain("_esni." + hostname),
		dns.TypeTXT,
	)
	if err != nil {
		//log.Print("Get ESNI keys of ", hostname, ": ", err.Error())
		return nil
	}
	for _, a := range rsp.Answer {
		if a.Type != typeTXT {
			continue
		}
		// Quad9 response is a quoted string
		data := strings.Trim(a.Data, " \t\"'")
		rawEsniKeys, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			log.Print("Error: DoH DecodeString: ", err.Error())
			continue
		}
		esniKeys, err := tris.ParseESNIKeys(rawEsniKeys)
		if err == nil {
			return esniKeys
		}
		log.Print("Error: DoH ParseESNIKeys: ", err.Error())
	}
	return nil
}

func renewEsniKeys(c *tris.Config) {
	if c == nil || c.ClientESNIKeys == nil {
		return
	}
	esniKeys := seekEsniKeys(c.ServerName)
	if esniKeys != nil {
		c.ClientESNIKeys = esniKeys
	}
}

func resolve(hostname string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	rsp, err := dohClient.Query(ctx, dns.Domain(hostname), dns.TypeA)
	if err != nil {
		return "", err
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
		return "", errors.New("DoH: No match was found for '" + hostname + "'")
	}
	return ips[rand.Intn(len(ips))], nil
}
