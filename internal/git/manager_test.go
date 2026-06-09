package git

import (
	"testing"
)

func TestRepoName(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{"https://github.com/org/my-project", "my-project"},
		{"https://github.com/org/my-project.git", "my-project"},
		{"git@github.com:org/repo.git", "repo"},
		{"", "repo"},
	}
	for _, c := range cases {
		got := repoName(c.url)
		if got != c.want {
			t.Errorf("repoName(%q) = %q, want %q", c.url, got, c.want)
		}
	}
}

func TestCleanup_Nil(t *testing.T) {
	m := NewManager("")
	if err := m.Cleanup(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
