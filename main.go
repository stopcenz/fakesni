package main

import (
    "flag"
    "fmt"
    "log"
    "os/exec"
    "runtime"
    "strings"
    "time"
)

// around https://github.com/ValdikSS/GoodbyeDPI/issues/71

const DEFAULT_HOST   = "rutracker.org"
const DEFAULT_PORT   = "443"
const DEFAULT_IP     = "195.82.146.214"
const FAKE_SNI       = "vk.com"
const LISTEN_ADDRESS = "127.0.0.1"
const LISTEN_PORT    = 56789

type appCfg struct {
	listenAddress  string
	listenPort     int
	remoteHostname string
	remoteIp       string
	remotePort     string
	fakeSni        string
	ignoreCert     bool
}

func runBrowser(u string, remoteHostname string) {
	platform := runtime.GOOS
	log.Print("The " + platform + " platform")
	duration := 2 * time.Second
	time.Sleep(duration)
	fmt.Println("Access to '" + remoteHostname + "' via the following URL: " + u)
	// https://dwheeler.com/essays/open-files-urls.html
	var cmd string
	var arg string
	if platform == "windows" {
		cmd = "cmd"
		arg = "/c start " + u
	} else if platform == "darwin" {
		cmd = "open"
		arg = u
	} else if platform == "linux" {
		cmd = "xdg-open"
		arg = u
	} else {
		return
	}
	err := exec.Command(cmd, arg).Start()
	if err != nil {
		log.Print("Error " + err.Error())
	}
}

func main() {
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
	ip, err := getIp(config.remoteHostname)
	if err != nil {
		fmt.Println("Can't resolve '" + config.remoteHostname + "':", err)
		if config.remoteHostname != DEFAULT_HOST  {
			panic("IP address not detected")
		}
		config.remoteIp = DEFAULT_IP
		fmt.Println("Using default IP - " + DEFAULT_IP)
	} else {
		config.remoteIp = ip
		log.Print("Found IP address for '" + config.remoteHostname + "' - " + ip)
	}
	h := config.listenAddress
	if h == "0.0.0.0" {
		h = "127.0.0.1"
	}
	u := fmt.Sprintf("http://%s:%d", h, config.listenPort)
	if *nobrowser {
		fmt.Println("Access to '" + config.remoteHostname + "' via the following URL: " + u)
	} else {
		go runBrowser(u, config.remoteHostname)
	}
	startServer(config)
}
