package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var allHosts string = ""
var bodyRe *regexp.Regexp
var domainRe *regexp.Regexp
var secureRe *regexp.Regexp = regexp.MustCompile(`(?i);\s*secure[^;]*`)
var hostRe *regexp.Regexp = regexp.MustCompile(`^([^:])(:([0-9]+)|)$`)
var trimPortRe *regexp.Regexp = regexp.MustCompile(`:.*`)

func convertInit(aliases Aliases) {
	if allHosts != "" {
		return
	}
	hosts := []string{}
	for _, alias := range aliases {
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

func convertHost(aliases Aliases, hostname string, port string) (string, bool) {
	hostname = strings.ToLower(hostname)
	if port == "" {
		port = "443"
	}
	for _, alias := range aliases {
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

func convertUrl(aliases Aliases, urlString string) string {
	u, err := url.Parse(urlString)
	if err != nil {
		return urlString
	}
	host, ok := convertHost(aliases, u.Hostname(), u.Port())
	if !ok {
		return urlString
	}
	if len(u.Scheme) > 0 {
		u.Scheme = "http"
	}
	u.Host = host
	return u.String()
}

func convertSetCookie(aliases  Aliases,
                      value    string,
                      r       *http.Request) string {

	host := r.Header.Get("Host")
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

func convertLocation(aliases Aliases, value string) string {
	return convertUrl(aliases, value)
}

func decode(response *http.Response) ([]byte, error) {
	encoding := response.Header.Get("Content-Encoding")
	re := regexp.MustCompile(`(?i)(^|,| )gzip($|,| )`)
	if !re.MatchString(encoding) {
		return ioutil.ReadAll(response.Body)
	}
	zr, err := gzip.NewReader(response.Body)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(zr)
	if err != nil {
		return nil, err
	}
	err = zr.Close()
	if err != nil {
		return nil, err
	}
	return body, nil
}

func convertHtml(aliases   Aliases,
                 response *http.Response,
                 w         http.ResponseWriter) {

	body, err := decode(response)
	if err != nil {
		log.Print("Error: convert html: " + err.Error())
		http.Error(w, err.Error(), 500)
		return
	}
	body = bodyRe.ReplaceAllFunc(body, func (b []byte) []byte {
		m := bodyRe.FindSubmatch(b)
		if len(m) < 2 {
			return b
		}
		head := m[1]
		u := m[4]
		tail := m[7]
		u = []byte(convertUrl(aliases, string(u)))
		return append(append(head, u...), tail...)
	})
	w.Header().Del("Content-Encoding")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.WriteHeader(response.StatusCode)
	_, err = io.Copy(w, bytes.NewReader(body))
	if err != nil {
		log.Print("Error: convert: " + err.Error())
	}
}

func convertBody(aliases   Aliases,
                 response *http.Response,
                 w         http.ResponseWriter) {

	contentType := strings.ToLower(response.Header.Get("Content-Type"))
	if strings.HasPrefix(contentType, "text/html") {
		convertHtml(aliases, response, w)
		return
	}
	w.WriteHeader(response.StatusCode)
	io.Copy(w, response.Body)
}

func convertHeader(aliases   Aliases,
                   response *http.Response,
                   w         http.ResponseWriter,
                   r        *http.Request) {

	for key, values := range response.Header {
		for _, value := range values {
			if key == "Location" {
				value = convertLocation(aliases, value)
			}
			if key == "Set-Cookie" {
				value = convertSetCookie(aliases, value, r)
			}
			if len(value) > 0 {
				w.Header().Add(key, value)
			}
		}
	}
}

func convertResponse(aliases   Aliases,
                     response *http.Response,
                     w         http.ResponseWriter,
                     r        *http.Request) {

	convertInit(aliases)
	convertHeader(aliases, response, w, r)
	convertBody(aliases, response, w)
}
