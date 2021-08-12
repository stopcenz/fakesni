package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

const READ_TO = 60 * time.Second
const WRITE_TO = 60 * time.Second

func fetch(alias  *Alias,
           client *http.Client,
           r      *http.Request) (*http.Response, error) {

	u := "https://" + alias.IP + ":" + alias.Port + r.URL.RequestURI()
	reqest, err := http.NewRequest(r.Method, u, r.Body)
	if err != nil {
		return nil, err
	}
	reqest.Header = r.Header
	reqest.Header.Del("Accept-Encoding")
	reqest.Host = alias.Hostname
	response, err := client.Do(reqest)
	//log.Printf("%s %s", r.Method, u)
	return response, err
}

func startServer(config *Config, srvIndex int, wg sync.WaitGroup) {
	alias := config.Aliases[srvIndex]
	// https://golang.org/src/crypto/tls/example_test.go
	tlsConfig := &tls.Config{
		// Set InsecureSkipVerify to skip the default validation we are
		// replacing. This will not disable VerifyConnection.
		InsecureSkipVerify: true,
		MaxVersion: tls.VersionTLS12,
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
	if config.TLS13 {
		tlsConfig.MaxVersion = tls.VersionTLS13
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
	redirectPolicyFunc := func (req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	client := &http.Client{Transport: tr, CheckRedirect: redirectPolicyFunc}
	handler := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		response, err := fetch(alias, client, r)
		if err != nil {
			log.Print("Error: ", err.Error())
			http.Error(w, "Proxy error\r\n" + err.Error(), 500)
			return
		}
		convertResponse(config.Aliases, response, w, r)
	})
	addr := fmt.Sprintf("%s:%d", config.ListenAddress, alias.ListenPort)
	log.Print("Start listening " + addr)
	server := &http.Server{
		Addr:           addr,
		Handler:        handler,
		ReadTimeout:    READ_TO,
		WriteTimeout:   WRITE_TO,
		MaxHeaderBytes: 0xffff,
	}
	host := alias.Hostname
	if alias.Port != "443" {
		host += ":" + alias.Port
	}
	log.Print(host + " (" + alias.IP + ") <--> " + alias.Addr)
	log.Print(server.ListenAndServe())
	wg.Done()
}
