package handlers

import "testing"

func TestDockerhub_NameAndTag(t *testing.T) {
	tests := []struct {
		name, tag string
		dockerhub Dockerhub
	}{
		{"myRepo", "v1", Dockerhub{
			Repository: DockerhubRepository{
				RepoName: "myRepo",
			},
			PushData: DockerhubPushData{
				Tag: "v1",
			},
		}},
	}

	for _, test := range tests {
		name, tag := test.dockerhub.NameAndTag()
		if name != test.name {
			t.Errorf("expected repo name: %s, got: %s", test.name, name)
		}
		if tag != test.tag {
			t.Errorf("expected tag: %s, got: %s", test.tag, tag)
		}
	}
}
