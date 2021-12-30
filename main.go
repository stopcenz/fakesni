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
	tris "github.com/stopcenz/tls-tris"
)

// around https://github.com/ValdikSS/GoodbyeDPI/issues/71

const DEFAULT_HOST   = "rutracker.org"
const DEFAULT_PORT   = "443"
const DEFAULT_IP     = "45.132.105.85"
const FAKE_SNI       = "vk.com"
const LISTEN_ADDRESS = "127.0.0.1"
const LISTEN_PORT    = 10000

var Version = "v0.0" // updated automatically

type Alias struct {
	Hostname   string  // "rutracker.org"
	Port       string  // "443"
	IP         string  // "195.82.146.214"
	ListenPort int     // 10000
	Addr       string  // "127.0.0.1:10001"
	Client     *http.Client
	EsniKeys   *tris.ESNIKeys
	TrisConfig *tris.Config
}

type Aliases []*Alias

type Config struct {
	ListenAddress string
	Aliases       Aliases
	FakeSNI       string
	IgnoreCert    bool
	NoBrowser     bool
}

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
	config := &Config{}
	listenAddress  := flag.String("addr", LISTEN_ADDRESS, "Local address. Set to 0.0.0.0 for listen all network interfaces.")
	listenPort     := flag.Int("port", LISTEN_PORT, "Port to run on.")
	flag.Var(&(config.Aliases), "host", "Remote host.")
	fakeSNI        := flag.String("sni", FAKE_SNI, "Value of fake SNI.")
	ignoreCert     := flag.Bool("ignorecert", false, "Skip certificate verification.")
	nobrowser      := flag.Bool("nobrowser", false, "Don't start browser.")
	flag.Parse()
	if *listenPort < 0 || *listenPort > 0xffff {
		log.Fatal(errors.New("Invalid port"))
	}
	if len(config.Aliases) < 1 {
		config.Aliases = Aliases{&Alias{
			Hostname : DEFAULT_HOST, 
			Port     : DEFAULT_PORT, 
			IP       : DEFAULT_IP,
		}}
	}
	config.ListenAddress = *listenAddress
	config.FakeSNI       = *fakeSNI
	config.IgnoreCert    = *ignoreCert
	config.NoBrowser     = *nobrowser
	if config.IgnoreCert {
		log.Print("Verify site certificate disabled")
	}
	local := *listenAddress
	if local == "0.0.0.0" {
		local = "127.0.0.1"
	}
	var wg sync.WaitGroup
	for n, alias := range config.Aliases {
		alias.ListenPort = *listenPort + n
		alias.Addr = fmt.Sprintf("%s:%d", local, alias.ListenPort)
		ip, esniKeys, err := getIp(alias.Hostname)
		
		if esniKeys == nil {
			log.Print("Using SNI value '" + config.FakeSNI + "' for " + alias.Hostname)
		} else {
			log.Print("Сonnect with ESNI to " + alias.Hostname)
			alias.EsniKeys = esniKeys
		}
		if err == nil {
			alias.IP = ip
		}
		if alias.IP == "" {
			log.Print(err.Error())
		} else {
			go startServer(config, n, wg)
			wg.Add(1)
		}
	}
	go browserStart(config)
	wg.Wait()
}
