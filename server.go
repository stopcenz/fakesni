package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"time"
)

const READ_TO = 60 * time.Second
const WRITE_TO = 60 * time.Second

func fetch(config *appCfg,
           client *http.Client,
           r      *http.Request) (*http.Response, error) {

	u := "https://" + config.remoteIp + ":" + config.remotePort + r.URL.RequestURI()
	reqest, err := http.NewRequest(r.Method, u, r.Body)
	if err != nil {
		return nil, err
	}
	reqest.Header = r.Header
	reqest.Host = config.remoteHostname
	response, err := client.Do(reqest)
	if err != nil {
		return nil, err
	}
	//log.Printf("%s %s %d", r.Method, u, response.StatusCode)
	return response, nil
}

func startServer(config *appCfg) {
	// https://golang.org/src/crypto/tls/example_test.go
	tlsConfig := &tls.Config{
		// Set InsecureSkipVerify to skip the default validation we are
		// replacing. This will not disable VerifyConnection.
		InsecureSkipVerify: true,
		ServerName:         config.fakeSni, // the magic is here
		VerifyConnection:   func(cs tls.ConnectionState) error {
			if config.ignoreCert {
				return nil
			}
			opts := x509.VerifyOptions{
				DNSName:       config.remoteHostname, // dafault is cs.ServerName,
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
		DisableCompression:     true, // disable unpack body
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
		response, err := fetch(config, client, r)
		if err != nil {
			log.Print("Error: ", err.Error())
			http.Error(w, "Proxy error\r\n" + err.Error(), 500)
			return
		}
		convertResponse(config, response, w)
	})
	addr := fmt.Sprintf("%s:%d", config.listenAddress, config.listenPort)
	log.Print("Listen " + addr)
  	server := &http.Server{
		Addr:           addr,
		Handler:        handler,
		ReadTimeout:    READ_TO,
		WriteTimeout:   WRITE_TO,
		MaxHeaderBytes: 0xffff,
	}
	log.Fatal(server.ListenAndServe())
}
