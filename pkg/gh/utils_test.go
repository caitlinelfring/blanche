package gh

import (
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/google/go-github/v31/github"
)

// This is based on how the github package handles their tests
func setup() (client *github.Client, mux *http.ServeMux, cleanup func()) {
	mux = http.NewServeMux()
	server := httptest.NewServer(mux)

	client = github.NewClient(nil)
	client.BaseURL, _ = url.Parse(server.URL + "/")

	return client, mux, server.Close
}
