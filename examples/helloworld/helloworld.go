package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

var port = flag.Int("port", 8080, "IP port")

func printenv(w http.ResponseWriter, _ *http.Request) {
	for _, e := range os.Environ() {
		fmt.Fprintf(w, "%s\n", e)
	}
}

func home(w http.ResponseWriter, _ *http.Request) {
	io.WriteString(w, "<html><body>Hello World!</body></html>")
}

func main() {
	flag.Parse()
	http.HandleFunc("/", home)
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, "ok\n")
	})
	http.HandleFunc("/env", printenv)
	fmt.Printf("Serving on port %d\n", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
