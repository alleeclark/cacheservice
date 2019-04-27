package main

import (
	"cacheservice"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
)

const (
	host     = "127.0.0.1"
	protocol = "http://"
)

var (
	serverURL string
	proxyURL  string
)

// change to read from config file
func init() {
	serverPort := 3000
	proxyPort := 8080
	flag.IntVar(&serverPort, "serverPort", serverPort, "Server Port")
	flag.IntVar(&proxyPort, "proxyPort", proxyPort, "Server Port")
	flag.Parse()
	serverURL = fmt.Sprintf("%s:%d", host, serverPort)
	proxyURL = fmt.Sprintf("%s:%d", host, proxyPort)
}

func main() {

	// make sure to skip clean
	router := mux.NewRouter().SkipClean(true)
	// add caching router
	router.HandleFunc("/cacheservice", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			cacheservice.GetFormCache(w, r)
		} else if r.Method == http.MethodPost {
			cacheservice.SaveToCache(w, r)
		} else if r.Method == http.MethodPut {
			cacheservice.UpdateCacheEntry(w, r)
		}
	})
	router.HandleFunc("/invalidatecache", cacheservice.InvalidateEntry)
	go cacheservice.PurgeCache()
	app := negroni.New()
	app.UseHandler(router)
	app.Run(":3000")
	log.Printf("CTRL+C to exit")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
