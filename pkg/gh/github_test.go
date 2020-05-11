package gh

import "testing"

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
