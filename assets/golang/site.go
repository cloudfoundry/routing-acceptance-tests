package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	http.HandleFunc("/", hello)
	port := os.Getenv("PORT")
	fmt.Printf("Listening on %s...", port)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: nil,
	}
	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func hello(res http.ResponseWriter, _ *http.Request) {
	fmt.Println("Received request ", time.Now())
	fmt.Fprintln(res, "go, world")
}
