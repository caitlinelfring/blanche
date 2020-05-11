package config

import (
	"os"
	"reflect"
	"testing"
)

func TestLoad(t *testing.T) {
	var manifests ManifestConfigs
	if err := manifests.load("manifest-test.yaml"); err != nil {
		t.Error(err)
	}

	if err := manifests.load("manifest-no.yaml"); err == nil {
		t.Error("should have returned an error, but got nil")
	}
}

func TestGetManifest(t *testing.T) {
	os.Setenv("MANIFEST_PATH", "manifest-test.yaml")

	tests := []struct {
		value    string
		expected *ManifestConfig
	}{
		{"celfring/guestbook", &ManifestConfig{
			DockerRepo: "celfring/guestbook",
			Manifests: []ManifestEntry{
				{File: "charts/guestbook/values.yaml", ConfigRepo: "caitlin615/argocd-demo", BaseBranch: "master", PullRequest: false},
			},
		}},
		{"celfring/no-entry", nil},
	}

	for _, test := range tests {
		m := GetManifest(test.value)
		if !reflect.DeepEqual(m, test.expected) {
			t.Errorf("expected: %+v got: %+v", test.expected, m)
		}
	}

	// Test GetManifest failure if manifest config file can't be found
	os.Setenv("MANIFEST_PATH", "manifest-no.yaml")
	manifests = nil
	m2 := GetManifest("celfring/guestbook")
	if m2 != nil {
		t.Errorf("expected nil manifest, got %v", m2)
	}

}

func TestGetEnvDefault(t *testing.T) {
	os.Setenv("__TEST", "got_it")
	if a := getEnvDefault("__TEST", "default_value"); a != "got_it" {
		t.Errorf("expected: %s, got: %s", "got_it", a)
	}
	os.Unsetenv("__TEST")
	if b := getEnvDefault("__TEST", "default_value"); b != "default_value" {
		t.Errorf("expected: %s, got: %s", "default_value", b)
	}
}

func TestGenerateGitUpdates(t *testing.T) {
	m := &ManifestConfig{
		DockerRepo: "celfring/guestbook",
		Manifests: []ManifestEntry{
			{File: "charts/guestbook/values.yaml", ConfigRepo: "caitlin615/argocd-demo", BaseBranch: "master", PullRequest: false},
		},
	}
	err := m.GenerateGitUpdates("celfring/guestbook", "notValidSemVer")
	if err != ErrTagNotValid {
		t.Errorf("expected error: %s, got: %s", ErrTagNotValid, err)
	}

	// TODO: This needs more tests
}

func TestParseRepo(t *testing.T) {
	tests := []struct {
		repo                        string
		expectedOwner, expectedRepo string
	}{
		{"celfring/guestbook", "celfring", "guestbook"},
		{"celfring/guestbook/not/used", "celfring", "guestbook"},
		{"celfring", "celfring", ""},
		{"", "", ""},
	}
	for _, test := range tests {
		o, r := parseRepo(test.repo)
		if o != test.expectedOwner {
			t.Errorf("expected owner: %s, got: %s", test.expectedOwner, o)
		}
		if r != test.expectedRepo {
			t.Errorf("expected repo: %s, got: %s", test.expectedRepo, r)
		}
	}
}
