package manifest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const FileName = ".personal-cloud.yaml"

type Manifest struct {
	Name    string       `yaml:"name"`
	Image   string       `yaml:"image"`
	Build   Build        `yaml:"build"`
	Service Service      `yaml:"service"`
	Route   Route        `yaml:"route"`
	Compose Compose      `yaml:"compose"`
	Test    string       `yaml:"test"`
}

type Build struct {
	Context    string `yaml:"context"`
	Dockerfile string `yaml:"dockerfile"`
}

type Service struct {
	Container  string `yaml:"container"`
	Port       int    `yaml:"port"`
	HealthPath string `yaml:"health_path"`
}

type Route struct {
	Exposure string `yaml:"exposure"` // public | private
	Host     string `yaml:"host"`
}

type Compose struct {
	Template string `yaml:"template"`
	EnvFile  string `yaml:"env_file"`
}

// Find walks from dir upward for .personal-cloud.yaml.
func Find(start string) (string, *Manifest, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", nil, err
	}
	for {
		path := filepath.Join(dir, FileName)
		if _, err := os.Stat(path); err == nil {
			m, err := Load(path)
			return dir, m, err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil, fmt.Errorf("%s not found from %s", FileName, start)
		}
		dir = parent
	}
}

func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	m.applyDefaults()
	return &m, nil
}

func (m *Manifest) applyDefaults() {
	if m.Compose.Template == "" {
		m.Compose.Template = "default"
	}
	if m.Route.Exposure == "" {
		m.Route.Exposure = "public"
	}
	if m.Service.HealthPath == "" {
		m.Service.HealthPath = "/health"
	}
	if m.Compose.EnvFile == "" && m.Name != "" {
		m.Compose.EnvFile = fmt.Sprintf("/opt/personal-cloud/apps/%s/.env", m.Name)
	}
}

func (m *Manifest) Validate(repoRoot string) error {
	var errs []error
	if m.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}
	if m.Image == "" {
		errs = append(errs, errors.New("image is required"))
	}
	if m.Build.Context == "" {
		errs = append(errs, errors.New("build.context is required"))
	}
	if m.Build.Dockerfile == "" {
		errs = append(errs, errors.New("build.dockerfile is required"))
	}
	if m.Service.Container == "" {
		errs = append(errs, errors.New("service.container is required"))
	}
	if m.Service.Port <= 0 {
		errs = append(errs, errors.New("service.port must be positive"))
	}
	exposure := strings.ToLower(m.Route.Exposure)
	if exposure != "public" && exposure != "private" {
		errs = append(errs, fmt.Errorf("route.exposure must be public or private, got %q", m.Route.Exposure))
	}
	if exposure == "public" && m.Route.Host == "" {
		errs = append(errs, errors.New("route.host is required when exposure is public"))
	}
	if m.Compose.Template != "default" && m.Compose.Template != "with-postgres" {
		errs = append(errs, fmt.Errorf("compose.template must be default or with-postgres, got %q", m.Compose.Template))
	}

	df := filepath.Join(repoRoot, m.Build.Dockerfile)
	if _, err := os.Stat(df); err != nil {
		errs = append(errs, fmt.Errorf("dockerfile not found: %s", df))
	}
	ctx := filepath.Join(repoRoot, m.Build.Context)
	if st, err := os.Stat(ctx); err != nil || !st.IsDir() {
		errs = append(errs, fmt.Errorf("build.context not found: %s", ctx))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// PrivateHost returns the default private route hostname.
func (m *Manifest) PrivateHost(tailnetBase string) string {
	base := strings.TrimPrefix(strings.TrimSpace(tailnetBase), ".")
	return fmt.Sprintf("%s.%s", m.Name, base)
}

// ResolvedHost returns route host for the given exposure override (empty = use manifest).
func (m *Manifest) ResolvedHost(exposure, tailnetBase string) string {
	if strings.ToLower(exposure) == "private" {
		if m.Route.Host != "" {
			return m.Route.Host
		}
		return m.PrivateHost(tailnetBase)
	}
	return m.Route.Host
}
