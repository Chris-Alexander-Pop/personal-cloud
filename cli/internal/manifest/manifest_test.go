package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndValidate(t *testing.T) {
	dir := t.TempDir()
	df := filepath.Join(dir, "Dockerfile")
	if err := os.WriteFile(df, []byte("FROM alpine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := filepath.Join(dir, "src")
	if err := os.MkdirAll(ctx, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, FileName)
	yml := `name: demo
image: ghcr.io/example/demo
build:
  context: src
  dockerfile: Dockerfile
service:
  container: demo
  port: 8080
route:
  exposure: public
  host: demo.example.com
`
	if err := os.WriteFile(path, []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Validate(dir); err != nil {
		t.Fatal(err)
	}
	if m.PrivateHost("ts.net") != "demo.ts.net" {
		t.Fatalf("private host: %s", m.PrivateHost("ts.net"))
	}
}

func TestParseAndValidateRemote(t *testing.T) {
	m, err := Parse([]byte(`name: music-serve
image: ghcr.io/example/music-serve
build:
  context: .
  dockerfile: Dockerfile
service:
  container: music-serve
  port: 8040
route:
  exposure: private
compose:
  template: with-music-stack
`))
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Validate(""); err != nil {
		t.Fatal(err)
	}
	gitServer, err := Parse([]byte(`name: private-git
image: ghcr.io/example/private-git
build:
  context: .
  dockerfile: Dockerfile
service:
  container: private-git
  port: 3000
  health_path: /api/healthz
route:
  exposure: private
compose:
  template: with-git-server
`))
	if err != nil {
		t.Fatal(err)
	}
	if err := gitServer.Validate(""); err != nil {
		t.Fatal(err)
	}
	if _, err := Parse([]byte("   \n")); err == nil {
		t.Fatal("expected empty manifest error")
	}
}
