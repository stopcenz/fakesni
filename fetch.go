package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"math/rand"
	"net"
	"net/http"
	"syscall"
	"time"
	tris "github.com/stopcenz/tls-tris"
	//tris "tris"
	"log"
)

func getTrisConfig(alias *Alias) *tris.Config {
	c := &tris.Config{
		ServerName: alias.Hostname,
		FakeServerName: config.FakeSNI,
		Padding: config.Padding,
		InsecureSkipVerify: config.IgnoreCert,
		ClientSessionCache: tris.NewLRUClientSessionCache(0),
	}
	if config.Esni {
		c.ClientESNIKeys = seekEsniKeys(alias.Hostname)
	}
	c.VerifyPeerCertificate = func(rawCerts [][]byte, 
			verifiedChains [][]*x509.Certificate) error {
		
		if config.IgnoreCert {
			return nil
		}
		opts := x509.VerifyOptions{
			DNSName: alias.Hostname,
			Intermediates: x509.NewCertPool(),
		}
		for _, rawCert := range rawCerts[1:] {
			certs, err := x509.ParseCertificates(rawCert)
			if err != nil {
				log.Print("Error: parse certificate: ", err.Error())
				continue
			}
			for _, cert := range certs {
				opts.Intermediates.AddCert(cert)
			}
		}
		cert0, err := x509.ParseCertificates(rawCerts[0])
		if err != nil {
			log.Print("Error: parse certificate: ", err.Error())
			return err
		}
		_, err = cert0[0].Verify(opts)
		return err
	}
	return c
}

func trisDial(alias *Alias) (*tris.Conn, error) {
	mute := alias.Mute
	alias.Mute = true
	trisConfig := alias.TrisConfig
	if (trisConfig == nil) {
		trisConfig = getTrisConfig(alias)
	}
	print := func(i ...interface{}) {
		if !mute {
			log.Print(i...)
		}
	}
	printDigest := func(n int) {
		if mute {
			return
		}
		if n > 0 {
			log.Print("Test #", n, " for '", alias.Hostname, "': Connect with extentions:")
		} else {
			log.Print("Connect to '", alias.Hostname, "' with extentions:")
		}
		if trisConfig.FakeServerName == "" {
			log.Print("  SNI: disabled,")
		} else {
			log.Print("  SNI: enabled, ServerName: '", trisConfig.FakeServerName, "',")
		}
		if trisConfig.ClientESNIKeys == nil {
			log.Print("  ESNI: disabled,")
		} else {
			log.Print("  ESNI: enabled, found keys,")
		}
		if trisConfig.Padding < 0 {
			log.Print("  TLS Padding: disabled.")
		} else {
			log.Print("  TLS Padding: ", trisConfig.Padding, " bytes.")
		}
	}
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
		Control: func(network, address string, c syscall.RawConn) error {
			print("Socket ready, handshaking...")
			return nil
		},	
	}
	if config.Esni && trisConfig.ClientESNIKeys == nil {
		print("ESNI keys not found for '", alias.Hostname, "'.")
	}
	nTest := 0
	// default parameters
	if config.Auto && alias.TrisConfig == nil {
		trisConfig.FakeServerName = ""
		trisConfig.ClientESNIKeys = nil
		trisConfig.Padding = -1
		nTest = 1
	}
	renewEsniKeys(trisConfig)
	printDigest(nTest)
	conn, err := tris.DialWithDialer(
		dialer,
		"tcp", 
		alias.IP + ":" + alias.Port, 
		trisConfig,
	)
	if err == nil {
		alias.TrisConfig = trisConfig
		return conn, nil
	}
	if !mute || trisConfig == alias.TrisConfig {
		log.Print("Connection failed: ", err.Error())
	}
	if !config.Auto {
		return nil, err
	}
	if trisConfig == alias.TrisConfig {
		log.Print("Reset configuration for '", alias.Hostname, "'.")
		alias.Mute = false
		alias.TrisConfig = nil
		return trisDial(alias)
	}
	// use ESNI
	esniKeys := seekEsniKeys(alias.Hostname)
	if esniKeys == nil {
		print("Test #2 for '", alias.Hostname, "': Error: ESNI keys not found.")
	} else {
		trisConfig.FakeServerName = ""
		trisConfig.ClientESNIKeys = esniKeys
		trisConfig.Padding = -1
		printDigest(2)
		conn, err := tris.DialWithDialer(
			dialer,
			"tcp", 
			alias.IP + ":" + alias.Port, 
			trisConfig,
		)
		if err == nil {
			alias.TrisConfig = trisConfig
			return conn, nil
		}
		print("Connection failed: ", err.Error())
	}
	// apply TLS Padding
	p := rand.Intn(999) + 12000
	trisConfig.FakeServerName = alias.Hostname
	trisConfig.ClientESNIKeys = nil
	trisConfig.Padding = p
	printDigest(3)
	conn, err = tris.DialWithDialer(
		dialer,
		"tcp", 
		alias.IP + ":" + alias.Port, 
		trisConfig,
	)
	if err == nil {
		alias.TrisConfig = trisConfig
		return conn, nil
	}
	print("Connection failed: ", err.Error())
	return nil, err
}

func trisFetch(alias    *Alias, 
		       fetchReq *http.Request,
		       w        http.ResponseWriter,
		       r        *http.Request) error {
	
	mute := alias.Mute
	conn, err := trisDial(alias)
	if err != nil {
		return err
	}
	// Maybe no conn.Close() call required. The http lib will do the right thing.
	defer conn.Close()
	if !mute {
		log.Print("Successful connection to '", alias.Hostname, "'.")
	}
	err = fetchReq.Write(conn)
	if err != nil {
		return err
	}
	fetchRes, err := http.ReadResponse(bufio.NewReader(conn), fetchReq)
	if err != nil {
		return err
	}
	defer fetchRes.Body.Close()
	convertResponse(config.Aliases, fetchRes, w, r)
	return nil
}

func httpFetch(alias  *Alias,
		fetchReq *http.Request, 
		w http.ResponseWriter, 
		r *http.Request) error {
		
	if alias.Client == nil {
		// https://golang.org/src/crypto/tls/example_test.go
		tlsConfig := &tls.Config{
			// Set InsecureSkipVerify to skip the default validation we are
			// replacing. This will not disable VerifyConnection.
			InsecureSkipVerify: true,
			ServerName: config.FakeSNI, // the magic is here
			VerifyConnection: func(cs tls.ConnectionState) error {
				if config.IgnoreCert {
					return nil
				}
				opts := x509.VerifyOptions{
					DNSName: alias.Hostname, // default is cs.ServerName,
					Intermediates: x509.NewCertPool(),
				}
				for _, cert := range cs.PeerCertificates[1:] {
					opts.Intermediates.AddCert(cert)
				}
				_, err := cs.PeerCertificates[0].Verify(opts)
				return err
			},
		}
		tr := &http.Transport{
			MaxIdleConns:           1,
			IdleConnTimeout:        10 * time.Second,
			TLSClientConfig:        tlsConfig,
			MaxResponseHeaderBytes: 0xffff,
			WriteBufferSize:        0xffff,
			ReadBufferSize:         0xffff,
			ForceAttemptHTTP2:      false,
		}
		rp := func (req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
		alias.Client = &http.Client{
			Transport: tr, 
			CheckRedirect: rp,
		}
	}
	fetchRes, err := alias.Client.Do(fetchReq)
	if err != nil {
		return err
	}
	defer fetchRes.Body.Close()
	convertResponse(config.Aliases, fetchRes, w, r)
	return nil
}

func fetchAndWrite(alias  *Alias,
		           w      http.ResponseWriter,
		           r      *http.Request) error {
           
	ip, err := resolve(alias.Hostname)
	if err != nil && alias.IP == "" {
		return err
	}
	if ip != "" {
		alias.IP = ip
	}
	u := "https://" + alias.IP + ":" + alias.Port + r.URL.RequestURI()
	fetchReq, err := http.NewRequest(r.Method, u, r.Body)
	if err != nil {
		return err
	}
	fetchReq.Header = r.Header.Clone()
	fetchReq.Header.Set("Accept-Encoding", "gzip, deflate")
	fetchReq.Header.Set("Connection", "close")
	fetchReq.Close = true
	if alias.Port == "443" {
		fetchReq.Host = alias.Hostname
	} else {
		fetchReq.Host = alias.Hostname + ":" + alias.Port
	}
	return trisFetch(alias, fetchReq, w, r)
}