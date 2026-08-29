package github

import (
	"fmt"
	"strings"
)

// NormalizeRepo turns "music-serve", "owner/music-serve", or a github.com URL
// into "owner/repo". Bare names use defaultOwner (github.owner in config).
func NormalizeRepo(raw, defaultOwner string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("repo is empty")
	}
	s = strings.TrimSuffix(s, ".git")
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "git@github.com:")
	s = strings.TrimPrefix(s, "github.com/")
	s = strings.Trim(s, "/")

	if !strings.Contains(s, "/") {
		owner := strings.TrimSpace(defaultOwner)
		if owner == "" {
			return "", fmt.Errorf("%q needs an owner (pass owner/repo or set github.owner)", raw)
		}
		if strings.ContainsAny(s, " \t") {
			return "", fmt.Errorf("invalid repo %q", raw)
		}
		return owner + "/" + s, nil
	}

	parts := strings.Split(s, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid repo %q (want owner/name)", raw)
	}
	if strings.ContainsAny(parts[0], " \t") || strings.ContainsAny(parts[1], " \t") {
		return "", fmt.Errorf("invalid repo %q", raw)
	}
	return parts[0] + "/" + parts[1], nil
}
