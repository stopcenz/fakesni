package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"github.com/likexian/doh-go"
	tris "github.com/stopcenz/tls-tris"
	//tris "tris"
)

// around https://github.com/ValdikSS/GoodbyeDPI/issues/71

const DEFAULT_HOST     = "rutracker.org"
const DEFAULT_PORT     = "443"
const DEFAULT_IP       = "45.132.105.85"
const LISTEN_ADDRESS   = "127.0.0.1"
const LISTEN_PORT      = 10000

var Version = "v0.0" // updated automatically

type Alias struct {
	Hostname   string  // "rutracker.org"
	Port       string  // "443"
	IP         string  // "195.82.146.214"
	ListenPort int     // 10000
	Addr       string  // "127.0.0.1:10001"
	Client     *http.Client
	TrisConfig *tris.Config
	Mute       bool
}

type Aliases []*Alias

type Config struct {
	ListenAddress string
	Aliases       Aliases
	FakeSNI       string
	Padding       int
	Esni          bool
	IgnoreCert    bool
	NoBrowser     bool
	Auto          bool
}
var config *Config
var dohClient *doh.DoH

func (aliases *Aliases) String() string {
	a := []string{}
	for _, alias := range *aliases {
		a = append(a, alias.Hostname + ":" + alias.Port)
    	}
    	return strings.Join(a, "\r\n")
}

func (alias *Aliases) Set(host string) error {
	hostRe := regexp.MustCompile(`^(https?:\/\/|)(.*@|)([-a-z0-9\.]+\.[a-z]{2,})(:([0-9]{1,5})|)$`)
	m := hostRe.FindStringSubmatch(strings.ToLower(host))
	if len(m) < 2 {
		return errors.New("Invalid host: '" + host + "'")
	}
	hostname := m[3]
	port := m[5]
	if port == "" {
		port = "443"
	}
	*alias = append(*alias, &Alias{Hostname: hostname, Port: port})
	return nil
}

func main() {
	log.Print("FakeSNI " + Version)
	config = &Config{}
	listenAddress  := flag.String("addr", LISTEN_ADDRESS, "Local address. Set to 0.0.0.0 for listen all network interfaces.")
	listenPort     := flag.Int("port", LISTEN_PORT, "Port to run on.")
	flag.Var(&(config.Aliases), "host", "Remote host. Optionally, you can specify the port number.")
	fakeSNI        := flag.String("sni", "", "Value for SNI. Not sent SNI by default.")
	padding        := flag.Int("padding", -1, "TLS Padding size (RFC 7685). An offset can be added before the SNI.")
	esni           := flag.Bool("esni", false, "Enable connections with ESNI.")
	ignoreCert     := flag.Bool("ignorecert", false, "Skip server certificate verification. Even the hostname is not validated.")
	nobrowser      := flag.Bool("nobrowser", false, "Don't start browser.")
	flag.Parse()
	if *listenPort < 0 || *listenPort > 0xffff {
		log.Fatal(errors.New("Invalid network port."))
	}
	if *padding < -1 || *padding > 0xffff {
		log.Fatal(errors.New("Invalid padding. Accepted from -1 to 65535."))
	}
	config.ListenAddress = *listenAddress
	config.FakeSNI       = *fakeSNI
	config.Padding       = *padding
	config.Esni          = *esni
	config.IgnoreCert    = *ignoreCert
	config.NoBrowser     = *nobrowser
	if len(config.Aliases) < 1 {
		config.Aliases = Aliases{&Alias{
			Hostname : DEFAULT_HOST, 
			Port     : DEFAULT_PORT, 
			IP       : DEFAULT_IP,
		}}
		log.Print("Parameters not specified. Example configuration applied:")
		log.Print("$ fakesni -host ", DEFAULT_HOST)
	}
	if config.IgnoreCert {
		log.Print("Insecure! Verify site certificate disabled.")
	}
	config.Auto = config.FakeSNI    == "" && 
                  config.Esni       == false &&
                  config.Padding    == -1 &&
                  config.IgnoreCert == false
	local := *listenAddress
	// init doh client, auto select the fastest provider
	dohClient = doh.Use(
		doh.CloudflareProvider, 
		doh.GoogleProvider, 
		doh.Quad9Provider,
	)
	dohClient.EnableCache(true)
	var wg sync.WaitGroup
	for n, alias := range config.Aliases {
		alias.ListenPort = *listenPort + n
		alias.Addr = fmt.Sprintf("%s:%d", local, alias.ListenPort)
		ip, err := resolve(alias.Hostname)
		if err != nil {
			log.Print(err.Error())
			continue
		}
		alias.IP = ip
		go startServer(n, wg)
		wg.Add(1)
	}
	go browserStart(config)
	wg.Wait()
}
