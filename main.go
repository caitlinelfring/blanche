package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/RentTheRunway/blanche/pkg/handlers"
	"github.com/gorilla/mux"
)

// These will be populated during a build
var (
	BuildVersion string = ""
	BuildTime    string = ""
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	r := mux.NewRouter()
	r.HandleFunc("/webhook/{type}", handlers.DockerHandler)
	r.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"buildTime": BuildTime, "buildVersion": BuildVersion})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	address := "0.0.0.0:" + port
	log.Println("now listening on", address)
	log.Fatal(http.ListenAndServe(address, r))
}
