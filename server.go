package main

import (
	//"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

const READ_TO = 120 * time.Second
const WRITE_TO = 120 * time.Second

type Context struct {
	Id string
	Alias *Alias
	IP string
	Request *http.Request
	ResponseWriter http.ResponseWriter
	HeadersSent bool
	DomainFronting string
	Mute bool
	Repeat bool
}

func (c *Context) WriteHeader(status int) {
	if !c.HeadersSent {
		c.HeadersSent = true
		c.ResponseWriter.WriteHeader(status)
	}
}

var connId uint64

func startServer(alias *Alias, wg sync.WaitGroup) {
	handler := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		connId += 1
		c := &Context{
			Id: fmt.Sprintf("%d", connId),
			Alias: alias,
			Request: r,
			ResponseWriter: w,
			Mute: alias.Mute,
		}
		alias.Mute = true
		if r.URL.RequestURI() == "/favicon.ico" {
			c.Mute = true
		}
		err := fetch(c)
		if err == nil {
			return
		}
		alias.Mute = false
		if c.HeadersSent {
			return
		}
		if c.Repeat {
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Location", "http://" + alias.Addr + r.URL.RequestURI())
			c.WriteHeader(307)
			w.Write([]byte(err.Error()))
		} else {
			http.Error(w, "Connection error: " + err.Error(), 500)
		}
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
	log.Print(alias.Host, " <--> ", alias.Addr)
	log.Print(server.ListenAndServe())
	wg.Done()
}

