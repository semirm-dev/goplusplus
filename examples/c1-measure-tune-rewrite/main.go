// 🌩 Go++ 🌩 #16 · Memory: Measure, tune, rewrite :: WIP
//
// A small service to practice on. See README.md for the walkthrough.
//
//	go run .              // naive report, no traffic
//	go run . -load        // naive report, with generated traffic
//	go run . -load -tuned // tuned report, with generated traffic
package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"sync/atomic"
	"time"
)

var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 64<<10)
		return &b
	},
}

func main() {
	tuned := flag.Bool("tuned", false, "serve the tuned report")
	load := flag.Bool("load", false, "generate traffic against /report")
	flag.Parse()

	orders := sampleOrders(2000)
	var served atomic.Int64

	app := http.NewServeMux()
	app.HandleFunc("/report", func(w http.ResponseWriter, r *http.Request) {
		served.Add(1)
		w.Header().Set("Content-Type", "text/csv")

		if *tuned {
			bp := bufPool.Get().(*[]byte)
			defer bufPool.Put(bp)
			*bp = ReportTuned(*bp, orders)
			w.Write(*bp)
			return
		}
		io.WriteString(w, ReportNaive(orders))
	})

	go func() {
		log.Println("pprof on http://localhost:6060/debug/pprof/")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	go func() {
		for range time.Tick(5 * time.Second) {
			log.Printf("served %d requests in the last 5s", served.Swap(0))
		}
	}()

	if *load {
		go generateLoad("http://localhost:8080/report", 4)
	}

	log.Printf("app on http://localhost:8080/report (tuned=%v, load=%v)", *tuned, *load)
	log.Fatal(http.ListenAndServe("localhost:8080", app))
}

func generateLoad(url string, n int) {
	time.Sleep(200 * time.Millisecond) // let the server start
	for range n {
		go func() {
			for {
				resp, err := http.Get(url)
				if err != nil {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}()
	}
}
