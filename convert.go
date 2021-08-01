package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var cookieRe = regexp.MustCompile(`(?i);\s*(domain=|secure)[^;]*`)

func convertSetCookie(config *appCfg, value string) string {
	value = cookieRe.ReplaceAllString(value, "")
	return value
}

func convertLocation(config *appCfg, value string) string {
	u, err := url.Parse(value)
	if err != nil {
		log.Print("Invalid location: " + value)
		return value
	}
	if strings.ToLower(u.Host) != config.remoteHostname {
		return value
	}
	value = u.RequestURI()
	if value == "" {
		value = "/"
	}
	return value
}

func convertResponse(config   *appCfg,
                     response *http.Response,
                     w         http.ResponseWriter) {

	for key, values := range response.Header {
		keyL := strings.ToLower(key)
		
		for _, value := range values {
			if keyL == "location" {
				value = convertLocation(config, value)
			}
			if keyL == "set-cookie" {
				value = convertSetCookie(config, value)
			}
			if len(value) > 0 {
				w.Header().Add(key, value)
			}
		}
	}
	w.WriteHeader(response.StatusCode)
	io.Copy(w, response.Body)
}
