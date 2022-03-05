package main

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

const MAX_HTML_BODY = 1e8

var allHosts string = ""
var bodyRe *regexp.Regexp
var domainRe *regexp.Regexp
var secureRe *regexp.Regexp = regexp.MustCompile(`(?i);\s*secure[^;]*`)
var hostRe *regexp.Regexp = regexp.MustCompile(`^([^:])(:([0-9]+)|)$`)
var trimPortRe *regexp.Regexp = regexp.MustCompile(`:.*`)

func convertInit() {
	if allHosts != "" {
		return
	}
	hosts := []string{}
	for _, alias := range config.Aliases {
		if alias.Port == "443" {
			hosts = append(hosts, alias.Hostname)
		}
		hosts = append(hosts, alias.Hostname + ":" + alias.Port)
	}
	allHosts = strings.Join(hosts, "|")
	domainRe = regexp.MustCompile(`(?i)(;\s*domain=\.?)(` + allHosts + ")[^;]*")
	bodyRe = regexp.MustCompile(`(?i)(\<[^\<\>]+\s(src|href|action)=('|"|))` +
	                            `((https?:|)\/\/(` + allHosts + `))([^a-z0-9])`)
}

func convertHost(hostname string, port string) (string, bool) {
	hostname = strings.ToLower(hostname)
	if port == "" {
		port = "443"
	}
	for _, alias := range config.Aliases {
		if hostname != alias.Hostname {
			continue
		}
		if port != alias.Port {
			continue
		}
		return alias.Addr, true
	}
	return "", false
}

func convertUrl(urlString string) string {
	u, err := url.Parse(urlString)
	if err != nil {
		return urlString
	}
	host, ok := convertHost(u.Hostname(), u.Port())
	if !ok {
		return urlString
	}
	if len(u.Scheme) > 0 {
		u.Scheme = "http"
	}
	u.Host = host
	return u.String()
}

func convertSetCookie(c *Context, value string) string {
	host := c.Request.Header.Get("Host")
	if host == "" {
		zeroHostRe := regexp.MustCompile(`(?i);\s*domain=[^;]*`)
		value = zeroHostRe.ReplaceAllString(value, "")
	} else {
		hostnameRe := regexp.MustCompile(`:.*`)
		hostname := hostnameRe.ReplaceAllString(host, "")
		value = domainRe.ReplaceAllString(value, "${1}" + hostname)
	}
	secureRe := regexp.MustCompile(`(?i);\s*secure[^;]*`)
	value = secureRe.ReplaceAllString(value, "")
	return value
}

func convertLocation(c *Context, value string) string {
	return convertUrl(value)
}

func decode(res *http.Response) ([]byte, error) {
	if res.Header.Get("Content-Length") == "0" {
		return []byte{}, nil
	}
	reader := res.Body
	ce := res.Header.Get("Content-Encoding")
	encodings := strings.Split(ce, ",")
	for i := len(encodings) - 1; i >= 0; i-- {
		switch strings.ToLower(strings.Trim(encodings[i], " \t")) {
			case "":
			case "identity":
			case "chunked":
				// chunked encoding is
				// automatically removed as necessary when receiving requests
			case "gzip":
				gr, err := gzip.NewReader(reader)
				if err != nil {
					log.Print("Error while decode '", ce, "'")
					return nil, err
				}
				defer gr.Close()
				reader = gr
			case "deflate":
				dr, err := zlib.NewReader(reader)
				if err != nil {
					log.Print("Error while decode '", ce, "'")
					return nil, err
				}
				defer dr.Close()
				reader = dr
			default:
				return nil, errors.New("Content-Encoding '" + ce + "' not supported.")
		}
	}
	return ioutil.ReadAll(io.LimitReader(reader, MAX_HTML_BODY))
}

func convertHtml(c *Context, res *http.Response) error {
	body, err := decode(res)
	if err != nil {
		return err
	}
	body = bodyRe.ReplaceAllFunc(body, func (b []byte) []byte {
		m := bodyRe.FindSubmatch(b)
		if len(m) < 2 {
			return b
		}
		head := m[1]
		u := m[4]
		tail := m[7]
		u = []byte(convertUrl(string(u)))
		return append(append(head, u...), tail...)
	})
	w := c.ResponseWriter
	w.Header().Del("Content-Encoding")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	c.WriteHeader(res.StatusCode)
	_, err = io.Copy(w, bytes.NewReader(body))
	if err != nil && DBG {
		s := res.Request.Method + " " + res.Request.URL.RequestURI()
		log.Print("Error: convert html: ", err.Error(), " - ", s)
	}
	return nil
}

func convertBody(c *Context, res *http.Response) error {
	contentType := strings.ToLower(res.Header.Get("Content-Type"))
	if strings.HasPrefix(contentType, "text/html") {
		return convertHtml(c, res)
	}
	// Preread first 1MB of body.
	// This is needed to handle the case where the connection was dropped.
	buf, err := ioutil.ReadAll(io.LimitReader(res.Body, 1e6))
	if err != nil {
		log.Print("Convert: Connection dropped: ", err.Error())
		return err
	}
	c.WriteHeader(res.StatusCode)
	_, err = c.ResponseWriter.Write(buf)
	if err == nil {
		_, err = io.Copy(c.ResponseWriter, res.Body)
	}
	if err != nil && DBG {
		log.Print("DBG: Error: convert body: ", err.Error())
		log.Print("  ", res.Request.Method, " ", c.Request.URL.RequestURI())
	}
	return nil
}

func convertHeader(c *Context, res *http.Response) {
	for key, values := range res.Header {
		for _, value := range values {
			if key == "Location" {
				value = convertLocation(c, value)
			}
			if key == "Set-Cookie" {
				value = convertSetCookie(c, value)
			}
			if len(value) > 0 {
				c.ResponseWriter.Header().Add(key, value)
			}
		}
	}
	if config.NoScript {
		c.ResponseWriter.Header().Add("Content-Security-Policy", "script-src 'none';")
	}
}

func convertResponse(c *Context, res *http.Response) error {
	convertInit()
	convertHeader(c, res)
	if res.Request.Method != http.MethodHead {
		return convertBody(c, res)
	}
	return nil
}

