package gh

import (
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
