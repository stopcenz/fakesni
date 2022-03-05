package main

import (
	"bufio"
	"crypto/x509"
	"net"
	"net/http"
	"syscall"
	tris "github.com/stopcenz/tls-tris"
	//tris "tris"
)

func getTrisConfig(c *Context) (*tris.Config, error) {
	trisConfig := &tris.Config{
		ServerName: c.Alias.Hostname,
		FakeServerName: c.Alias.FakeSNI,
		Padding: c.Alias.Padding,
		InsecureSkipVerify: config.IgnoreCert,
		ClientSessionCache: tris.NewLRUClientSessionCache(0),
	}
	if c.Alias.ESNI {
		esniKeys, err := seekESNIKeys(c.Alias.Hostname)
		if err != nil {
			return nil, err
		}
		trisConfig.ClientESNIKeys = esniKeys
		if !c.Mute {
			logPrint(c, "Found ESNI keys for '", c.Alias.Hostname, "'")
		}
	}
	trisConfig.VerifyPeerCertificate = func(rawCerts [][]byte, 
			verifiedChains [][]*x509.Certificate) error {
		
		if config.IgnoreCert {
			return nil
		}
		opts := x509.VerifyOptions{
			DNSName: c.Alias.Hostname,
			Intermediates: x509.NewCertPool(),
		}
		for _, rawCert := range rawCerts[1:] {
			certs, err := x509.ParseCertificates(rawCert)
			if err != nil {
				logPrint(c, "Error: parse certificate: ", err.Error())
				continue
			}
			for _, cert := range certs {
				opts.Intermediates.AddCert(cert)
			}
		}
		cert0, err := x509.ParseCertificates(rawCerts[0])
		if err != nil {
			logPrint(c, "Error: parse certificate: ", err.Error())
			return err
		}
		_, err = cert0[0].Verify(opts)
		return err
	}
	return trisConfig, nil
}

func trisDial(c *Context) (*tris.Conn, *tris.Config, error) {
	var err error
	trisConfig := c.Alias.TrisConfig
	if trisConfig == nil {
		trisConfig, err = getTrisConfig(c)
		if err != nil {
			return nil, nil, err
		}
	}
	dialer := &net.Dialer{
		Timeout: config.Timeout,
		Control: func(network, address string, rc syscall.RawConn) error {
			if !c.Mute {
				logPrint(c, "Socket ready, handshaking...")
			}
			return nil
		},
	}
	conn, err := tris.DialWithDialer(
		dialer,
		"tcp", 
		c.IP + ":" + c.Alias.Port, 
		trisConfig,
	)
	return conn, trisConfig, err
}

func trisRequest(c *Context, req *http.Request) error {
	conn, trisConfig, err := trisDial(c)
	if err == nil {
		defer conn.Close()
		req.Close = true
		err = req.Write(conn)
		if err == nil {
			res, err := http.ReadResponse(bufio.NewReader(conn), req)
			if err == nil {
				defer res.Body.Close()
				err = convertResponse(c, res)
			}
		}
	}
	if err == nil {
		c.Alias.TrisConfig = trisConfig
	}
	return err
}

