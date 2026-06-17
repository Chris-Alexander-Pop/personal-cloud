package env

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/your-org/personal-cloud/cli/internal/config"
	"github.com/your-org/personal-cloud/cli/internal/manifest"
)

const envSubdir = "env"

// PushResult describes a successful env push.
type PushResult struct {
	Local   string
	Remote  string
	SSHHost string
}

// Dir returns ~/.config/pc/env (mode 0700 when created by Init).
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, config.ConfigDir, envSubdir), nil
}

// LocalPath returns ~/.config/pc/env/<app>.env.
func LocalPath(appName string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appName+".env"), nil
}

// Init copies apps/<name>/.env.example from the personal-cloud clone to the
// local secrets file. Returns the destination path.
func Init(m *manifest.Manifest, cfg *config.Config, force bool) (string, error) {
	dest, err := LocalPath(m.Name)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(dest); err == nil && !force {
		return "", fmt.Errorf("%s already exists (use --force to overwrite)", dest)
	}

	src, err := examplePath(m.Name, cfg)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return "", err
	}
	if err := copyFile(src, dest); err != nil {
		return "", err
	}
	if err := os.Chmod(dest, 0o600); err != nil {
		return "", err
	}
	return dest, nil
}

func examplePath(appName string, cfg *config.Config) (string, error) {
	root, err := personalCloudRoot(cfg)
	if err != nil {
		return "", err
	}
	src := filepath.Join(root, "apps", appName, ".env.example")
	if _, err := os.Stat(src); err != nil {
		return "", fmt.Errorf("missing %s — add apps/%s/.env.example in personal-cloud", src, appName)
	}
	return src, nil
}

func personalCloudRoot(cfg *config.Config) (string, error) {
	root := strings.TrimSpace(cfg.PersonalCloud.LocalPath)
	if root == "" {
		root = os.Getenv("PERSONAL_CLOUD_ROOT")
	}
	if root == "" {
		return "", fmt.Errorf("set personal_cloud.local_path in ~/.config/pc/config.yaml (or PERSONAL_CLOUD_ROOT) to your personal-cloud clone")
	}
	return expandHome(root), nil
}

// ExampleSource returns the .env.example path for display (init summary).
func ExampleSource(appName string, cfg *config.Config) (string, error) {
	return examplePath(appName, cfg)
}

// Push uploads the local env file to the VM via scp (SSH config Host alias).
func Push(m *manifest.Manifest, cfg *config.Config, localFile string) (*PushResult, error) {
	sshHost := strings.TrimSpace(cfg.VM.SSH)
	if sshHost == "" {
		return nil, fmt.Errorf("vm.ssh is required in ~/.config/pc/config.yaml (SSH config Host alias, e.g. deploy)")
	}

	if localFile == "" {
		var err error
		localFile, err = LocalPath(m.Name)
		if err != nil {
			return nil, err
		}
	}
	if _, err := os.Stat(localFile); err != nil {
		return nil, fmt.Errorf("local env file not found: %s (run: pc env init)", localFile)
	}

	remote := m.Compose.EnvFile
	if remote == "" {
		remote = fmt.Sprintf("/opt/personal-cloud/apps/%s/.env", m.Name)
	}
	remoteDir := path.Dir(remote)

	if err := run("ssh", sshHost, "mkdir", "-p", remoteDir); err != nil {
		return nil, fmt.Errorf("ssh mkdir: %w", err)
	}
	dest := sshHost + ":" + remote
	if err := run("scp", localFile, dest); err != nil {
		return nil, fmt.Errorf("scp: %w", err)
	}
	if err := run("ssh", sshHost, "chmod", "600", remote); err != nil {
		return nil, fmt.Errorf("ssh chmod: %w", err)
	}

	return &PushResult{
		Local:   localFile,
		Remote:  remote,
		SSHHost: sshHost,
	}, nil
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func expandHome(p string) string {
	if p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return p
	}
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
