package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/RentTheRunway/blanche/pkg/config"
	"github.com/gorilla/mux"
)

type DockerRegistryHandler interface {
	NameAndTag() (name string, tag string)
}

func DockerHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	registryType := vars["type"]

	var dockerHandler DockerRegistryHandler
	switch registryType {
	case "dockerhub":
		dockerHandler = new(Dockerhub)
		// case "artifactory":
		// 	dockerHandler := Artifactory{}
	}
	if err := json.NewDecoder(r.Body).Decode(dockerHandler); err != nil {
		log.Println(err)
		return
	}
	name, tag := dockerHandler.NameAndTag()

	if match := config.GetManifest(name); match != nil {
		match.GenerateGitUpdates(name, tag)
	} else {
		log.Printf("No matching manifest for %s:%s", name, tag)
	}
	w.WriteHeader(http.StatusOK)
}
