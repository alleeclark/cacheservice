package cacheservice

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"
)

type cacheEntry struct {
	data       []byte
	expiration time.Time
}

var (
	cache  = make(map[string]*cacheEntry)
	mutex  = sync.RWMutex{}
	tickCh = time.Tick(10 * time.Minute)
)

var maxAgeRexexp = regexp.MustCompile(`maxage=(\d+)`)

//GetFormCache returns value
func GetFormCache(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
	mutex.RLock()
	defer mutex.RUnlock()
	key := r.URL.Query().Get("key")
	log.Printf("Searching cache for %s...", key)
	if entry, ok := cache[key]; ok {
		log.Println("found")
		w.Write(entry.data)
		return
	}
	w.WriteHeader(http.StatusNotFound)
	log.Println("Not Found")
}

//InvalidateEntry purges cache keys and values
func InvalidateEntry(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()
	key := r.URL.Query().Get("key")
	log.Printf("purging entry with key '%s'\n", key)
	delete(cache, key)
}

//SaveToCache POST key and value to cache
func SaveToCache(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
	mutex.Lock()
	defer mutex.Unlock()
	key := r.URL.Query().Get("key")
	cacheHeader := r.Header.Get("cache-control")
	if cacheHeader == "" || cacheHeader == "no-cache" {
		cacheHeader = "20"
	}
	dur, err := strconv.Atoi(cacheHeader)
	if err != nil {
		log.Fatal(err)
		dur = 20
	}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatalf("Unable to read response %s", err)
	}
	cache[key] = &cacheEntry{data: data, expiration: time.Now().Add(time.Duration(dur) * time.Minute)}
	log.Printf("Saving cache entry with key '%s' for %d minutes\n", key, dur)
	defer r.Body.Close()
}

//UpdateCacheEntry will update the entry time
func UpdateCacheEntry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
	mutex.Lock()
	defer mutex.Unlock()
	key := r.URL.Query().Get("key")
	cacheHeader := r.Header.Get("cache-control")
	log.Printf("Updating cache entry with key '%s' for %s seconds\n", key, cacheHeader)

	if cacheHeader == "" {
		cacheHeader = "20"
	}
	dur, err := strconv.Atoi(cacheHeader)
	if err != nil {
		log.Fatalf("Could not covnert cache header %s", err)
		dur = 20
	}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatalf("Error unable to read body %s", err)
	}
	if string(cache[key].data) != string(data) {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	cache[key] = &cacheEntry{data: data, expiration: time.Now().Add(time.Duration(dur-1) * time.Minute)}
	defer func() {
		io.Copy(ioutil.Discard, r.Body)
		r.Body.Close()
	}()
}

//PurgeCache purges the cache that is before the expiration date
func PurgeCache() {
	for range tickCh {
		mutex.Lock()
		now := time.Now()
		log.Println("purging cache")
		for k, v := range cache {
			if now.Before(v.expiration) {
				log.Printf("purging entry with key '%s'\n", k)
				delete(cache, k)
			}
		}
		mutex.Unlock()
	}
}
