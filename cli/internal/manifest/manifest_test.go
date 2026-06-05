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
