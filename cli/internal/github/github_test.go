package github

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/your-org/personal-cloud/cli/internal/config"
)

func TestNormalizeRepo(t *testing.T) {
	cases := []struct {
		raw, owner, want string
	}{
		{"music-serve", "chris", "chris/music-serve"},
		{"chris/music-serve", "", "chris/music-serve"},
		{"https://github.com/chris/music-serve.git", "", "chris/music-serve"},
		{"github.com/chris/music-serve", "", "chris/music-serve"},
		{"git@github.com:chris/music-serve.git", "", "chris/music-serve"},
	}
	for _, tc := range cases {
		got, err := NormalizeRepo(tc.raw, tc.owner)
		if err != nil {
			t.Fatalf("%s: %v", tc.raw, err)
		}
		if got != tc.want {
			t.Fatalf("%s: got %s want %s", tc.raw, got, tc.want)
		}
	}
	if _, err := NormalizeRepo("music-serve", ""); err == nil {
		t.Fatal("expected owner required")
	}
}

func TestAccessToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GHCR_TOKEN", "")
	cfg := &config.Config{}
	cfg.GitHub.Token = "from-cfg"
	if AccessToken(cfg) != "from-cfg" {
		t.Fatal("config token")
	}
	cfg.GitHub.Token = ""
	cfg.GHCR.Token = "ghcr"
	if AccessToken(cfg) != "ghcr" {
		t.Fatal("ghcr fallback")
	}
}

func TestResolve(t *testing.T) {
	yml := []byte(`name: music-serve
image: ghcr.io/chris/music-serve
build:
  context: .
  dockerfile: Dockerfile
service:
  container: music-serve
  port: 8080
route:
  exposure: private
compose:
  template: with-music-stack
`)
	encoded := base64.StdEncoding.EncodeToString(yml)

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/chris/music-serve", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, "auth", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"default_branch": "main"})
	})
	mux.HandleFunc("/repos/chris/music-serve/commits/main", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"sha": "abcdef1234567890"})
	})
	mux.HandleFunc("/repos/chris/music-serve/contents/.personal-cloud.yaml", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ref") != "abcdef1234567890" {
			http.Error(w, "bad ref", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"encoding": "base64",
			"content":  encoded,
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := New("tok", srv.URL)
	res, err := c.Resolve("chris/music-serve", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Manifest.Name != "music-serve" {
		t.Fatalf("name: %s", res.Manifest.Name)
	}
	if res.Git.FullSHA != "abcdef1234567890" || res.Git.ShortSHA != "abcdef1" {
		t.Fatalf("git: %+v", res.Git)
	}
	if err := res.Manifest.Validate(""); err != nil {
		t.Fatal(err)
	}
}
