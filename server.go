package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

const READ_TO = 60 * time.Second
const WRITE_TO = 60 * time.Second

func startServer(config *Config, srvIndex int, wg sync.WaitGroup) {
	alias := config.Aliases[srvIndex]
	handler := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		response, err := fetch(config, alias, r)
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
