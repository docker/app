package main

import (
	"fmt"
	"log"
	"net/http"
)

var (
	// Message is the text to serve
	Message string
)

func main() {
	if Message == "" {
		Message = "<h1>Hello from Docker App build demo!</h1>"
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Serve request from: %s\n", r.RemoteAddr)
		fmt.Fprintf(w, Message)
	})
	log.Fatal(http.ListenAndServe(":5000", nil))
}
