package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/your-org/personal-cloud/cli/internal/config"
	"github.com/your-org/personal-cloud/cli/internal/manifest"
)

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if got := expandHome("~"); got != home {
		t.Fatalf("~: got %q want %q", got, home)
	}
	if got := expandHome("~/foo"); got != filepath.Join(home, "foo") {
		t.Fatalf("~/foo: got %q", got)
	}
	if got := expandHome("/abs/path"); got != "/abs/path" {
		t.Fatalf("abs: got %q", got)
	}
}

func TestInitFromExample(t *testing.T) {
	pcRoot := t.TempDir()
	appDir := filepath.Join(pcRoot, "apps", "demo")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	example := filepath.Join(appDir, ".env.example")
	if err := os.WriteFile(example, []byte("FOO=bar\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)

	m := &manifest.Manifest{Name: "demo"}
	cfg := &config.Config{
		PersonalCloud: config.Personal{LocalPath: pcRoot},
	}

	dest, err := Init(m, cfg, false)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "FOO=bar\n" {
		t.Fatalf("content: %q", data)
	}
	info, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode: %o", info.Mode().Perm())
	}

	_, err = Init(m, cfg, false)
	if err == nil {
		t.Fatal("expected error when file exists")
	}

	dest2, err := Init(m, cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	if dest2 != dest {
		t.Fatalf("dest: %s vs %s", dest2, dest)
	}
}

func TestExamplePathRequiresRoot(t *testing.T) {
	_, err := examplePath("demo", &config.Config{})
	if err == nil {
		t.Fatal("expected error without local_path")
	}
}
