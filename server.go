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

func startServer(srvIndex int, wg sync.WaitGroup) {
	alias := config.Aliases[srvIndex]
	handler := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		err := fetchAndWrite(alias, w, r)
		if err != nil {
			log.Print("Error: server: ", err.Error())
			http.Error(w, "Connection error \r\n" + err.Error(), 500)
			return
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
	host := alias.Hostname
	if host == "0.0.0.0" {
		host = "localhost"
	}
	if alias.Port != "443" {
		host += ":" + alias.Port
	}
	log.Print(host + " (" + alias.IP + ") <--> " + alias.Addr)
	log.Print(server.ListenAndServe())
	wg.Done()
}
