package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/RentTheRunway/blanche/pkg/config"
	"github.com/RentTheRunway/blanche/pkg/webhook"
)

var manifests config.ManifestConfigs

// These will be populated during a build
var (
	BuildVersion string = ""
	BuildTime    string = ""
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	if err := manifests.Load(getEnvDefault("MANIFEST_PATH", "manifest.yaml")); err != nil {
		panic(err)
	}

	http.HandleFunc("/webhook", dhWebhookHandler)
	http.HandleFunc("/ping", pingHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	address := "0.0.0.0:" + port
	log.Println("now listening on", address)
	log.Fatal(http.ListenAndServe(address, nil))
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf(`{"buildTime": "%s", "buildVersion": "%s"}`, BuildTime, BuildVersion)))
}

func dhWebhookHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	dhResp, err := webhook.UnmarshalDHHook(body)
	if err != nil {
		log.Println(err, string(body))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if match := manifests.GetManifest(dhResp.Repository.RepoName); match != nil {
		match.GenerateGitUpdates(dhResp.Repository.RepoName, dhResp.PushData.Tag)
	} else {
		log.Printf("No matching manifest for %s:%s", dhResp.Repository.RepoName, dhResp.PushData.Tag)
	}
	// Don't need to tell dockerhub whether or not we were successful, it doesn't care
	w.WriteHeader(http.StatusOK)
}

// the following are helper functions for parsing environment variables
func getEnvDefault(key, defaultValue string) string {
	if v, ok := os.LookupEnv(key); ok && len(v) > 0 {
		return v
	}
	return defaultValue
}
