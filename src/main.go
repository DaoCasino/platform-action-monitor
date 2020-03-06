package main

import (
	"flag"
	"log"
	"net/http"
)

var addr = flag.String("addr", ":8888", "http service address")

func main() {
	flag.Parse()
	scraper := newScraper()
	go scraper.run()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveWs(scraper, w, r)
	})
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
