package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"time"
	tris "github.com/stopcenz/tls-tris"
)

func noVerifyCert(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	return nil
}

func fetchWithEsni(config *Config,
                   alias   *Alias,
                   request *http.Request) (*http.Response, error) {

	if (alias.TrisConfig == nil) {
		alias.TrisConfig = &tris.Config{
			ServerName: alias.Hostname,
			InsecureSkipVerify: config.IgnoreCert,
			ClientSessionCache: tris.NewLRUClientSessionCache(0),
			ClientESNIKeys: alias.EsniKeys,
		}
		if config.IgnoreCert {
			alias.TrisConfig.VerifyPeerCertificate = noVerifyCert
		}
	}
	
	dialer := new(net.Dialer)
	dialer.Timeout = 60 * time.Second
	conn, err := tris.DialWithDialer(
		dialer,
		"tcp", 
		alias.IP + ":" + alias.Port, 
		alias.TrisConfig,
	)
	if err != nil {
		return nil, err
	}
	err = request.Write(conn)
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(conn)
	return http.ReadResponse(reader, request)
}


func fetch(config *Config,
           alias  *Alias,
           r      *http.Request) (*http.Response, error) {
           
        u := "https://" + alias.IP + ":" + alias.Port + r.URL.RequestURI()
	request, err := http.NewRequest(r.Method, u, r.Body)
	if err != nil {
		return nil, err
	}
	request.Proto = "HTTP/1.1"
	request.Header = r.Header
	request.Header.Set("Accept-Encoding", "gzip")
	request.Host = alias.Hostname
	
	if alias.EsniKeys != nil {
		return fetchWithEsni(config, alias, request)
	}
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
	response, err := alias.Client.Do(request)
	//log.Printf("f %s %s", r.Method, u)
	return response, err
}