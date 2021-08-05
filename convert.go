package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)


func convertSetCookie(config *appCfg,
                      value   string,
                      r      *http.Request) string {

	host := r.Header.Get("Host")
	if host == "" {
		host = "127.0.0.1"
	}			
	hostRe := regexp.MustCompile(`^([^:]*).*`)
	hostname := hostRe.ReplaceAllString(host, "${1}")
	domainRe := regexp.MustCompile(`(?i)(;\s*domain=\.?)` + config.remoteHostname)
	value = domainRe.ReplaceAllString(value, "${1}" + hostname)
	secureRe := regexp.MustCompile(`(?i);\s*secure[^;]*`)
	value = secureRe.ReplaceAllString(value, "")
	return value
}


func convertLocation(config *appCfg, value string) string {
	u, err := url.Parse(value)
	if err != nil {
		log.Print("convert: invalid location: " + value)
		return value
	}
	if strings.ToLower(u.Hostname()) != config.remoteHostname {
		return value
	}
	if u.Port() != "" && u.Port() != config.remotePort {
		return value
	}
	u.Scheme = "http"
	u.Host = fmt.Sprintf("%s:%d", config.listenAddress, config.listenPort)
	return u.String()
}


func convertBody(config   *appCfg,
                 response *http.Response,
                 w         http.ResponseWriter) {

	contentType := strings.ToLower(response.Header.Get("Content-Type"))
	if !strings.HasPrefix(contentType, "text/html") {
		w.WriteHeader(response.StatusCode)
		io.Copy(w, response.Body)
		return
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Print("Error: convert: " + err.Error())
		w.Write([]byte(err.Error()))
		return
	}
	
	bodyRe := regexp.MustCompile(`(?i)(\<[^\<\>]+\s(src|href|action)=('|"|))` +
	                             `(https?:|)\/\/(` + 
	                             regexp.QuoteMeta(config.remoteHostname) + `|` + 
	                             regexp.QuoteMeta(config.remoteIp) +
	                             `)(:` + config.remotePort + `|)([^a-z0-9])`)
	mask := fmt.Sprintf("${1}http://%s:%d${7}", config.listenAddress, config.listenPort)
	body = bodyRe.ReplaceAll(body, []byte(mask))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.WriteHeader(response.StatusCode)
	_, err = w.Write(body)
	if err != nil {
		log.Print("Error: convert: " + err.Error())
	}
}


func convertResponse(config   *appCfg,
                     response *http.Response,
                     w         http.ResponseWriter,
                     r        *http.Request) {

	for key, values := range response.Header {
		for _, value := range values {
			if key == "Location" {
				value = convertLocation(config, value)
			}
			if key == "Set-Cookie" {
				value = convertSetCookie(config, value, r)
			}
			if len(value) > 0 {
				w.Header().Add(key, value)
			}
		}
	}
	convertBody(config, response, w)
}
