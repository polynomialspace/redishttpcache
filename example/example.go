package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	servertiming "github.com/mitchellh/go-server-timing"
	cache "github.com/polynomialspace/redishttpcache"
)

func main() {
	rdb := redis.NewClient(&redis.Options{
		//Password:   "password",
		//DB:         0,
		Addr:         "localhost:6379",
		ReadTimeout:  1 * time.Minute,
		WriteTimeout: 1 * time.Minute,
	})
	cachecfg := &cache.Config{
		Rdb:        rdb,
		Expiration: 5 * time.Second,
		ErrCallback: func(err error, r *http.Request) {
			log.Printf("%s: %v\n", r.URL.EscapedPath(), err)
		},
		HitCallback: func(r *http.Request) {
			log.Printf("hit: %s\n", r.URL.EscapedPath())
		},
		MissCallback: func(r *http.Request) {
			log.Printf("miss: %s\n", r.URL.EscapedPath())
		},
		CacheRequest: func(r *http.Request) bool {
			_, nocache := r.URL.Query()["nocache"]
			if nocache {
				log.Printf("%s: requested nocache", r.URL.EscapedPath())
			}
			return r.Method == "GET" && !nocache // eg: GET /foo?nocache
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// a perfect example of something you shouldn't cache :)
		fmt.Fprintf(w, "hello %s, the time is %v", r.RemoteAddr, time.Now())
	})
	log.Fatalln(http.ListenAndServe(":8180", servertiming.Middleware(cache.Middleware(mux, cachecfg), nil)))
}
