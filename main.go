package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
	"github.com/likexian/doh-go"
)

// around https://github.com/ValdikSS/GoodbyeDPI/issues/71

const DBG = false

const DEFAULT_HOST     = "rutracker.org"
const DEFAULT_PORT     = "443"
const DEFAULT_IPS      = "185.234.247.237|5.104.224.16|185.154.12.197"
const LISTEN_ADDRESS   = "127.0.0.1"
const LISTEN_PORT      = 10000

const DOMAIN_FRONTING = ".appspot.com|.blogspot.com|.herocuapp.com"
const WHITES = "vk.com|ok.ru|mail.ru|yandex.ru"

var Version = "v0.0" // updated automatically

type Config struct {
	ListenAddress string
	Aliases       Aliases
	FakeSNI       string
	ESNI          bool
	Padding       int
	NoScript      bool
	IgnoreCert    bool
	NoBrowser     bool
	Timeout       time.Duration
	Auto          bool
}

var config *Config
var dohClient *doh.DoH

func isAuto() bool {
	auto := true
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "sni" || f.Name == "esni" ||
				f.Name == "padding" || f.Name == "ignorecert" {
			auto = false
		}
	})
	return auto
}

func main() {
	log.Print("FakeSNI " + Version)
	config = &Config{}
	listenAddress  := flag.String("addr", LISTEN_ADDRESS, "Local address. Set to 0.0.0.0 for listen all network interfaces.")
	listenPort     := flag.Int("port", LISTEN_PORT, "Port to run on.")
	flag.Var(&(config.Aliases), "host", "Remote host. Optionally, you can specify the port number.")
	fakeSNI        := flag.String("sni", "", "Value for SNI. Not sent SNI by default.")
	esni           := flag.Bool("esni", false, "Enable connections with ESNI.")
	padding        := flag.Int("padding", -1, "TLS Padding size (RFC 7685). An offset can be added before the SNI.")
	timeout        := flag.Int("timeout", 5, "TLS handshake timeout in seconds.")
	noscript       := flag.Bool("noscript", false, "Disable JavaScript in HTML.")
	ignoreCert     := flag.Bool("ignorecert", false, "Skip server certificate verification.")
	nobrowser      := flag.Bool("nobrowser", false, "Don't start browser.")
	flag.Parse()
	if flag.NArg() > 0 {
		flag.PrintDefaults()
		return
	}
	if *listenPort < 0 || *listenPort > 0xffff {
		log.Fatal(errors.New("Invalid network port."))
	}
	if *padding < -1 || *padding > 0xffff {
		log.Fatal(errors.New("Invalid padding. Accepted from -1 to 65535."))
	}
	config.ListenAddress = *listenAddress
	config.FakeSNI       = *fakeSNI
	config.Padding       = *padding
	config.ESNI          = *esni
	config.NoScript      = *noscript
	config.IgnoreCert    = *ignoreCert
	config.NoBrowser     = *nobrowser
	config.Timeout       = time.Duration(*timeout) * time.Second
	if len(config.Aliases) < 1 {
		config.Aliases = Aliases{&Alias{
			Host     : DEFAULT_HOST,
			Hostname : DEFAULT_HOST,
			Port     : DEFAULT_PORT,
		}}
		log.Print("Parameter '-host' not specified. Example configuration applied.")
	}
	if config.IgnoreCert {
		log.Print("Insecure! Verify site certificate disabled.")
	}
	config.Auto = isAuto()
	local := *listenAddress
	// init doh client, auto select the fastest provider
	dohClient = doh.Use(
		doh.CloudflareProvider,
		doh.GoogleProvider,
		doh.Quad9Provider,
	)
	dohClient.EnableCache(true)
	rand.Seed(time.Now().UnixNano())
	var wg sync.WaitGroup
	for n, alias := range config.Aliases {
		alias.ListenPort = *listenPort + n
		alias.Addr = fmt.Sprintf("%s:%d", local, alias.ListenPort)
		go startServer(alias, wg)
		wg.Add(1)
	}
	go browserStart()
	wg.Wait()
}

