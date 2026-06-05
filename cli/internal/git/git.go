package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type Info struct {
	Root     string
	Remote   string // owner/repo
	Branch   string
	FullSHA  string
	ShortSHA string
	Dirty    bool
}

func Discover(repoRoot string) (*Info, error) {
	branch, err := run(repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}
	full, err := run(repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	short, err := run(repoRoot, "rev-parse", "--short", "HEAD")
	if err != nil {
		return nil, err
	}
	remote, err := run(repoRoot, "remote", "get-url", "origin")
	if err != nil {
		return nil, fmt.Errorf("git remote origin: %w", err)
	}
	ownerRepo, err := parseGitHubRemote(remote)
	if err != nil {
		return nil, err
	}
	status, _ := run(repoRoot, "status", "--porcelain")
	return &Info{
		Root:     repoRoot,
		Remote:   ownerRepo,
		Branch:   branch,
		FullSHA:  full,
		ShortSHA: short,
		Dirty:    strings.TrimSpace(status) != "",
	}, nil
}

func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), strings.TrimSpace(out.String()), err)
	}
	return strings.TrimSpace(out.String()), nil
}

func parseGitHubRemote(url string) (string, error) {
	url = strings.TrimSpace(url)
	url = strings.TrimSuffix(url, ".git")
	switch {
	case strings.HasPrefix(url, "git@github.com:"):
		return url[len("git@github.com:"):], nil
	case strings.Contains(url, "github.com"):
		// https://github.com/owner/repo
		parts := strings.Split(url, "github.com/")
		if len(parts) < 2 {
			break
		}
		slug := strings.Trim(parts[1], "/")
		if slug != "" {
			return slug, nil
		}
	}
	return "", fmt.Errorf("not a github.com remote: %s", url)
}
