package gh

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/google/go-github/v31/github"
)

func TestUpdateImageTag(t *testing.T) {
	tests := []struct {
		value    string
		expected string
	}{
		{"image:\n  tag: v1\n", "image:\n  tag: v2\n"},
		{"image:\n  tag: v1\n  repo: myRepo\n", "image:\n  repo: myRepo\n  tag: v2\n"},
	}

	for _, test := range tests {
		got, err := updateImageTag(test.value, "v2")
		if err != nil {
			t.Error(err)
		}

		if test.expected != got {
			t.Errorf("expected: %s, got: %s", test.expected, got)
		}
	}
}

func TestIsOlderVersionBumpPR(t *testing.T) {
	tests := []struct {
		image, tag string
		pr         *github.PullRequest
		expected   bool
	}{
		{"imageName", "v2", &github.PullRequest{Title: github.String("[auto-release] imageName:v1 for foo")}, true},
		{"imageName", "v2.2", &github.PullRequest{Title: github.String("[auto-release] imageName:v1.1 for foo")}, true},
		{"imageName", "v1.30", &github.PullRequest{Title: github.String("[auto-release] imageName:v1.3 for foo")}, true},
		{"imageName", "v1", &github.PullRequest{Title: github.String("[auto-release] imageName:v2 for foo")}, false},
		{"imageName", "latest", &github.PullRequest{Title: github.String("[auto-release] imageName:v2 for foo")}, false},
		{"imageName", "v1", &github.PullRequest{Title: github.String("[auto-release] imageName:latest for foo")}, false},
		{"imageName", "v1", &github.PullRequest{Title: github.String("[auto-release] differentImageName:latest for foo")}, false},
	}

	for _, test := range tests {
		got := isOlderVersionBumpPR(test.image, test.tag, test.pr)
		if got != test.expected {
			t.Errorf("isOlderVersionBumpPR(%s, %s, %v) | expected: %t, got: %t", test.image, test.tag, test.pr, test.expected, got)
		}
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		a, b     string
		expected bool
	}{
		{"v1", "v2", false},
		{"v2", "v1", true},
		{"latest", "v2", false},
		{"v1", "latest", false},
		{"notSemVer", "alsoNotSemVer", false},
	}

	for _, test := range tests {
		got := isNewerVersion(test.a, test.b)
		if got != test.expected {
			t.Errorf("isNewerVersion(%s, %s) | expected: %t, got: %t", test.a, test.b, test.expected, got)
		}
	}
}

func TestGitUpdate_createRef(t *testing.T) {
	client, mux, cleanup := setup()
	defer cleanup()

	mux.HandleFunc("/repos/o/r/git/refs/heads/master", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `
		  {
		    "ref": "refs/heads/master",
		    "url": "https://api.github.com/repos/o/r/git/refs/heads/master",
		    "object": {
		      "type": "commit",
		      "sha": "aa218f56b14c9653891f9e74264a383fa43fefbd",
		      "url": "https://api.github.com/repos/o/r/git/commits/aa218f56b14c9653891f9e74264a383fa43fefbd"
		    }
			}`)
	})

	mux.HandleFunc("/repos/o/r/git/refs", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `
		  {
		    "ref": "refs/heads/auto-release/master/o/r-v1",
		    "url": "https://api.github.com/repos/o/r/git/refs/heads/auto-release/master/o/r-v1",
		    "object": {
		      "type": "commit",
		      "sha": "aa218f56b14c9653891f9e74264a383fa43fefbd",
		      "url": "https://api.github.com/repos/o/r/git/commits/aa218f56b14c9653891f9e74264a383fa43fefbd"
		    }
		  }`)
	})

	g := newGitUpdate()
	g.client = client
	ref, err := g.createRef(g.baseRef, g.targetRef)
	if err != nil {
		t.Error(err)
	}
	expected := &github.Reference{
		Ref: github.String("refs/heads/auto-release/master/o/r-v1"),
		URL: github.String("https://api.github.com/repos/o/r/git/refs/heads/auto-release/master/o/r-v1"),
		Object: &github.GitObject{
			Type: github.String("commit"),
			SHA:  github.String("aa218f56b14c9653891f9e74264a383fa43fefbd"),
			URL:  github.String("https://api.github.com/repos/o/r/git/commits/aa218f56b14c9653891f9e74264a383fa43fefbd"),
		}}
	if !reflect.DeepEqual(ref, expected) {
		t.Errorf("Git.CreateRef expected %+v, got %+v", expected, ref)
	}
}

func TestGitUpdate_getManifestFileContents(t *testing.T) {
	client, mux, cleanup := setup()
	defer cleanup()

	mux.HandleFunc("/repos/o/r/contents/charts/r/values.yaml", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
		  "type": "file",
		  "name": "values.yaml",
			"path": "charts/r/values.yaml",
			"encoding": "base64",
			"content": "aW1hZ2U6CiAgdGFnOiB2MQo="
		}`)
	})

	g := newGitUpdate()
	g.client = client

	ref := &github.Reference{Ref: github.String("refs/heads/master")}
	content, err := g.getManifestFileContents(ref)
	if err != nil {
		t.Error(err)
	}
	expected := "image:\n  tag: v1\n"
	if expected != content {
		t.Errorf("expected %s, got %s", expected, content)
	}
}

func TestGitUpdate_newTreeWithChanges(t *testing.T) {
	client, mux, cleanup := setup()
	defer cleanup()

	mux.HandleFunc("/repos/o/r/git/trees", func(w http.ResponseWriter, r *http.Request) {
		v := new(github.Tree)
		if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
			t.Error(err)
		}
		expected := github.Tree{
			Entries: []*github.TreeEntry{
				{
					Path:    github.String("charts/r/values.yaml"),
					Mode:    github.String("100644"),
					Type:    github.String("blob"),
					Content: github.String("image:\n  tag: v2\n"),
				},
			},
		}
		if !reflect.DeepEqual(v.Entries, expected.Entries) {
			t.Errorf("expected: %+v, got: %+v", expected, v)
		}

		fmt.Fprint(w, `{
		  "sha": "5c6780ad2c68743383b740fd1dab6f6a33202b11",
		  "url": "https://api.github.com/repos/o/r/git/trees/5c6780ad2c68743383b740fd1dab6f6a33202b11",
		  "tree": [
		    {
			  "mode": "100644",
			  "type": "blob",
			  "sha":  "aad8feacf6f8063150476a7b2bd9770f2794c08b",
			  "path": "charts/r/values.yaml",
			  "url": "https://api.github.com/repos/o/r/git/blobs/aad8feacf6f8063150476a7b2bd9770f2794c08b"
		    }
		  ]
		}`)
	})

	mux.HandleFunc("/repos/o/r/contents/charts/r/values.yaml", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
		  "type": "file",
		  "name": "values.yaml",
			"path": "charts/r/values.yaml",
			"encoding": "base64",
			"content": "aW1hZ2U6CiAgdGFnOiB2MQo="
		}`)
	})

	g := newGitUpdate()
	g.client = client

	ref := &github.Reference{Ref: github.String("refs/heads/tree")}
	tree, err := g.newTreeWithChanges(ref)
	if err != nil {
		t.Error(err)
	}

	entry := tree.Entries[0]
	if entry.GetPath() != g.ManifestFile {
		t.Errorf("expected tree entry file: %s, got: %s", g.ManifestFile, entry.GetPath())
	}
}

func TestGitUpdate_pushCommit(t *testing.T) {
	client, mux, cleanup := setup()
	defer cleanup()
	// GetCommit
	mux.HandleFunc("/repos/o/r/commits/5c6780ad2c68743383b740fd1dab6f6a33202b11", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"sha":"5c6780ad2c68743383b740fd1dab6f6a33202b11",
			"commit": {
				"author": {
					"name": "Caitlin Elfring",
					"email": "celfring@gmail.com",
					"date": "2020-05-05T20:48:19Z"
				},
				"committer": {
					"name": "Caitlin Elfring",
					"email": "celfring@gmail.com",
					"date": "2020-05-05T20:48:19Z"
				},
				"tree": {
					"sha": "8ad54bad2954bfc7c86b398809925cf2c9169d89",
					"url": "https://api.github.com/repos/o/r/git/trees/8ad54bad2954bfc7c86b398809925cf2c9169d89"
				}
			},
			"parents": [
				{
					"sha": "c1c30f42271d4fa9ab79acaa9a64478772243d78"
				}
			]
		}`)
	})

	// CreateCommit
	mux.HandleFunc("/repos/o/r/git/commits", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"sha":"newCommitSha"}`)
	})

	// UpdateRef
	mux.HandleFunc("/repos/o/r/git/refs/heads/tree", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `
		  {
		    "ref": "refs/heads/tree",
		    "url": "https://api.github.com/repos/o/r/git/refs/heads/tree",
		    "object": {
		      "type": "commit",
		      "sha": "aa218f56b14c9653891f9e74264a383fa43fefbd",
		      "url": "https://api.github.com/repos/o/r/git/commits/aa218f56b14c9653891f9e74264a383fa43fefbd"
		    }
		  }`)
	})

	g := newGitUpdate()
	g.client = client

	ref := &github.Reference{Ref: github.String("refs/heads/tree"), Object: &github.GitObject{SHA: github.String("5c6780ad2c68743383b740fd1dab6f6a33202b11")}}
	tree := &github.Tree{
		Entries: []*github.TreeEntry{
			{
				Path: github.String("charts/r/values.yaml"),
				Mode: github.String("100644"),
				Type: github.String("blob"),
			},
		},
	}
	if err := g.pushCommit(ref, tree); err != nil {
		t.Error(err)
	}
}

func TestGitUpdate_createPR(t *testing.T) {
	client, mux, cleanup := setup()
	defer cleanup()

	expected := &github.NewPullRequest{
		Title: github.String("[auto-release] o/r:v2 for master"),
		Head:  github.String("refs/heads/auto-release/master/o/r-v2"),
		Base:  github.String("refs/heads/master"),
		Body:  github.String("This PR was automatically generated by a creation of a new docker image: o/r:v2"),
	}

	mux.HandleFunc("/repos/o/r/pulls", func(w http.ResponseWriter, r *http.Request) {
		got := new(github.NewPullRequest)
		json.NewDecoder(r.Body).Decode(got)

		if !reflect.DeepEqual(got, expected) {
			t.Errorf("expected %+v, got %+v", expected, got)
		}

		fmt.Fprint(w, `{
			"number":1,
			"html_url": "https://github.com/o/r/pulls/1",
			"title": "[auto-release] o/r:v2 for master",
			"body": "This PR was automatically generated by a creation of a new docker image: o/r:v2",
			"base": {"ref": "refs/heads/master"},
			"head": {"ref": "refs/heads/auto-release/master/o/r-v2"}
		}`)
	})

	g := newGitUpdate()
	g.client = client

	url, err := g.createPR(g.targetRef, g.baseRef)
	if err != nil {
		t.Error(err)
	}
	expectedUrl := "https://github.com/o/r/pulls/1"
	if url != expectedUrl {
		t.Errorf("expected: %s, got: %s", url, expectedUrl)
	}
}

func newGitUpdate() *gitUpdate {
	return NewGitUpdates(
		"o",
		"r",
		"charts/r/values.yaml",
		"master",
		"o/r",
		"v2",
		true,
		true)
}
