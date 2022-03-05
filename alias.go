package main

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	tris "github.com/stopcenz/tls-tris"
	//tris "tris"
)

func seekDomainFronting(hostname string) string {
	d := strings.ToLower(strings.Replace(DOMAIN_FRONTING, `.`, `\.`, -1))
	re := regexp.MustCompile(`^([-0-9a-z]+)(` + d + `)$`)
	m := re.FindAllStringSubmatch(strings.ToLower(hostname), 1)
	if len(m) > 0 && len(m[0]) > 2 {
		return m[0][2]
	}
	return ""
}

type Alias struct {
	Host           string    // "site.com" or "site.com:1234" if port not "443"
	Hostname       string    // "rutracker.org"
	Port           string    // "443"
	IPs            []string  // ["195.82.146.214", "45.132.105.85"]
	ListenPort     int       // 10000
	Addr           string    // "127.0.0.1:10001"
	Client         *http.Client
	TrisConfig     *tris.Config
	DomainFronting string
	FakeSNI        string
	ESNI           bool
	Padding        int
	Mute           bool
	Phase          int
}

type Aliases []*Alias

func (a *Aliases) String() string {
	arr := []string{}
	for _, alias := range *a {
		arr = append(arr, alias.Hostname + ":" + alias.Port)
	}
	return strings.Join(arr, "\r\n")
}

func (a *Aliases) Set(host string) error {
	hostRe := regexp.MustCompile(`^(https?:\/\/|)(.*@|)([-a-z0-9\.]+\.[a-z]{2,})(:([0-9]{1,5})|)$`)
	m := hostRe.FindStringSubmatch(strings.ToLower(host))
	if len(m) < 2 {
		return errors.New("Invalid host: '" + host + "'")
	}
	alias := Alias{
		Hostname: strings.ToLower(m[3]),
		Port: m[5],
	}
	if alias.Port == "" {
		alias.Port = "443"
	}
	if alias.Port == "443" {
		alias.Host = alias.Hostname
	} else {
		alias.Host = alias.Hostname + ":" + alias.Port
	}
	alias.IPs = []string{}
	alias.DomainFronting = seekDomainFronting(alias.Hostname)
	*a = append(*a, &alias)
	return nil
}

