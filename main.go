package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", indexHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}
