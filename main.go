package main

import (
    "flag"
    "fmt"
    "log"
    "strings"
)

// around https://github.com/ValdikSS/GoodbyeDPI/issues/71

const DEFAULT_HOST   = "rutracker.org"
const DEFAULT_PORT   = "443"
const DEFAULT_IP     = "195.82.146.214"
const FAKE_SNI       = "vk.com"
const LISTEN_ADDRESS = "127.0.0.1"
const LISTEN_PORT    = 56789

var Version = "v0.0" // updated automatically

type appCfg struct {
	listenAddress  string
	listenPort     int
	remoteHostname string
	remoteIp       string
	remotePort     string
	fakeSni        string
	ignoreCert     bool
}

func main() {
	log.Print("Starting FakeSNI " + Version)
	listenAddress  := flag.String("addr", LISTEN_ADDRESS, "Local address. Set to 0.0.0.0 for listen all network interfaces")
	listenPort     := flag.Int("port", LISTEN_PORT, "Port to run on")
	remoteHostname := flag.String("host", DEFAULT_HOST, "Remote hostname")
	fakeSni        := flag.String("sni", FAKE_SNI, "Value of fake SNI")
	ignoreCert     := flag.Bool("ignorecert", false, "Don't verify server certificate")
	nobrowser      := flag.Bool("nobrowser", false, "Don't start browser")
	flag.Parse()

	config := &appCfg{
		listenAddress:  *listenAddress,
		listenPort:     *listenPort,
		remoteHostname: strings.ToLower(*remoteHostname),
		remotePort:     DEFAULT_PORT,
		fakeSni:        *fakeSni,
		ignoreCert:     *ignoreCert,
	}
	log.Print("Using SNI value '" + config.fakeSni + "'")
	if config.ignoreCert {
		log.Print("Verify site certificate disabled")
	}
	ip, err := getIp(config.remoteHostname)
	if err != nil {
		if config.remoteHostname != DEFAULT_HOST  {
			log.Fatal(err)
		}
		config.remoteIp = DEFAULT_IP
		log.Print("Can't resolve '" + config.remoteHostname + "':", err)
		log.Print("Using default IP - " + DEFAULT_IP)
	} else {
		config.remoteIp = ip
		log.Print("Found IP address for '" + config.remoteHostname + "' - " + ip)
	}
	h := config.listenAddress
	if h == "0.0.0.0" {
		h = "127.0.0.1"
	}
	u := fmt.Sprintf("http://%s:%d", h, config.listenPort)
	prompt := "\r\nAccess to '" + config.remoteHostname + "' via the following URL: " + u
	if *nobrowser {
		fmt.Println(prompt)
	} else {
		go runBrowser(u, prompt)
	}
	startServer(config)
}
