package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

func logPrint(c *Context, i ...interface{}) {
	i = append([]interface{}{"[", c.Id, "] "}, i...)
	log.Print(i...)
}

func ptintDigest(c *Context) {
	if c.Mute {
		return
	}
	if config.Auto {
		logPrint(c, "Test #", c.Alias.Phase + 1, " for '", c.Alias.Hostname, "':")
	} else {
		logPrint(c, "Connect to '", c.Alias.Hostname, "':")
	}
	if c.Alias.FakeSNI != "" {
		logPrint(c, "  SNI: '", c.Alias.FakeSNI, "',")
	}
	if c.Alias.ESNI {
		logPrint(c, "  ESNI: enabled,")
	}
	if c.Alias.Padding >= 0 {
		logPrint(c, "  Padding: ", c.Alias.Padding, " bytes,")
	}
}

func printResult(c *Context, err error) {
	if err == nil {
		if !c.Mute {
			logPrint(c, "Successful connection with '", c.Alias.Hostname, "'")
		}
	} else {
		c.Alias.Client = nil
		c.Alias.TrisConfig = nil
		if c.Request.URL.RequestURI() != "/favicon.ico" {
			logPrint(c, "Error: ", err.Error())
		}
	}
}

func fetchRequest(c *Context, req *http.Request) error {
	var err error
	ptintDigest(c)
	if c.Alias.ESNI || c.Alias.Padding >= 0 {
		err = trisRequest(c, req)
		printResult(c, err)
		return err
	}
	client := c.Alias.Client
	if client == nil {
		// https://golang.org/src/crypto/tls/example_test.go
		tlsConfig := &tls.Config{
			// Set InsecureSkipVerify to skip the default validation we are
			// replacing. This will not disable VerifyConnection.
			InsecureSkipVerify: true,
			ServerName: c.Alias.FakeSNI, // the magic is here
			VerifyConnection: func(cs tls.ConnectionState) error {
				if config.IgnoreCert {
					return nil
				}
				opts := x509.VerifyOptions{
					DNSName: c.Alias.Hostname, // default is cs.ServerName,
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
			TLSHandshakeTimeout:    config.Timeout,
			IdleConnTimeout:        60 * time.Second,
			TLSClientConfig:        tlsConfig,
			MaxResponseHeaderBytes: 0xffff,
			WriteBufferSize:        0xffff,
			ReadBufferSize:         0xffff,
			ForceAttemptHTTP2:      false,
		}
		rp := func (req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
		client = &http.Client{
			Transport: tr, 
			CheckRedirect: rp,
		}
	}
	res, err := client.Do(req)
	if err == nil {
		defer res.Body.Close()
		err = convertResponse(c, res)
	}
	if err == nil {
		c.Alias.Client = client
	}
	printResult(c, err)
	return err
}

func fetchRequestAuto(c *Context, req *http.Request) error {
	var err error
	c.Repeat = true
	if c.Alias.Client != nil || c.Alias.TrisConfig != nil {
		err = fetchRequest(c, req)
		if err != nil {
			c.Alias.Phase = 0
			c.Alias.Mute = false
		}
		return err
	}
	if c.Alias.DomainFronting != "" {
		c.Alias.FakeSNI = randString(6, 16) + c.Alias.DomainFronting
		c.Alias.ESNI = false
		c.Alias.Padding = -1
	} else if c.Alias.Phase == 0 {			// Test #1
		c.Alias.FakeSNI = genWhiteHost()
		c.Alias.ESNI = false
		c.Alias.Padding = -1
	} else if c.Alias.Phase == 1 {			// Test #2
		c.Alias.FakeSNI = ""
		c.Alias.ESNI = true
		c.Alias.Padding = -1
	} else if c.Alias.Phase == 2 {			// Test #3
		// more https://habr.com/ru/post/477696/
		c.Alias.FakeSNI = genWhiteHost()
		c.Alias.ESNI = true
		c.Alias.Padding = -1
	} else if c.Alias.Phase == 3 {			// Test #4
		c.Alias.FakeSNI = c.Alias.Hostname
		c.Alias.ESNI = false
		c.Alias.Padding = rand.Intn(999) + 12000
	} else if c.Alias.Phase == 4 {			// Test #5
		c.Alias.FakeSNI = c.Alias.Hostname
		c.Alias.ESNI = false
		c.Alias.Padding = -1
	} else if c.Alias.Phase == 5 {			// Test #6
		c.Alias.FakeSNI = c.Alias.Hostname
		c.Alias.ESNI = false
		c.Alias.Padding = rand.Intn(999) + 13000
		c.Repeat = false
	} else {
		if DBG {
			logPrint(c, "Error: Fetch: Unknown phase: ", c.Alias.Phase)
		}
		c.Alias.Phase = 0
		return fetchRequestAuto(c, req)
	}
	err = fetchRequest(c, req)
	if err != nil {
		c.Alias.Phase += 1
	}
	return err
}

func fetch(c *Context) error {
	err := resolve(c)
	if err != nil {
		if c.Alias.Hostname != DEFAULT_HOST {
			return err
		}
		logPrint(c, err)
		c.Alias.IPs = strings.Split(DEFAULT_IPS, "|")
		resolve(c)
	}
	u := "https://" + c.IP + ":" + c.Alias.Port + c.Request.URL.RequestURI()
	if DBG {
		logPrint(c, c.Request.Method, " ", u)
	}
	rc, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(rc, c.Request.Method, u, c.Request.Body)
	if err != nil {
		logPrint(c, "Fetch error: ", err.Error())
		return err
	}
	req.Header = c.Request.Header.Clone()
	req.Header.Del("Range")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Host = c.Alias.Hostname
	if config.Auto {
		return fetchRequestAuto(c, req)
	}
	c.Alias.FakeSNI = config.FakeSNI
	c.Alias.ESNI = config.ESNI
	c.Alias.Padding = config.Padding
	return fetchRequest(c, req)
}

