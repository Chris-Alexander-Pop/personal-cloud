package github

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/your-org/personal-cloud/cli/internal/config"
	"github.com/your-org/personal-cloud/cli/internal/git"
	"github.com/your-org/personal-cloud/cli/internal/manifest"
)

const (
	defaultAPI = "https://api.github.com"
	userAgent  = "personal-cloud-pc"
	fileName   = manifest.FileName
)

// Client talks to the GitHub REST API to resolve an app repo for ship.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// Resolved is a manifest plus the commit Woodpecker should clone.
type Resolved struct {
	Repo     string
	Ref      string
	Branch   string
	Manifest *manifest.Manifest
	Git      *git.Info
}

// AccessToken returns a GitHub token from config or the environment.
// Public repos work without one; private repos need a token with contents:read.
func AccessToken(cfg *config.Config) string {
	if cfg != nil {
		if t := strings.TrimSpace(cfg.GitHub.Token); t != "" {
			return t
		}
		if t := strings.TrimSpace(cfg.GHCR.Token); t != "" {
			return t
		}
	}
	for _, k := range []string{"GITHUB_TOKEN", "GH_TOKEN", "GHCR_TOKEN"} {
		if t := strings.TrimSpace(os.Getenv(k)); t != "" {
			return t
		}
	}
	return ""
}

// New returns a client. baseURL empty means https://api.github.com.
func New(token, baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultAPI
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Resolve fetches .personal-cloud.yaml and the commit SHA for ref
// (branch, tag, or SHA). Empty ref uses the repo default branch.
func (c *Client) Resolve(ownerRepo, ref string) (*Resolved, error) {
	ownerRepo = strings.TrimSpace(ownerRepo)
	if ownerRepo == "" {
		return nil, fmt.Errorf("github repo is required")
	}
	ref = strings.TrimSpace(ref)

	meta, err := c.repo(ownerRepo)
	if err != nil {
		return nil, err
	}
	if ref == "" {
		ref = meta.DefaultBranch
	}
	if ref == "" {
		return nil, fmt.Errorf("github.com/%s: no default branch", ownerRepo)
	}

	commit, err := c.commit(ownerRepo, ref)
	if err != nil {
		return nil, err
	}
	body, err := c.file(ownerRepo, commit.SHA, fileName)
	if err != nil {
		return nil, err
	}
	m, err := manifest.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("github.com/%s %s: %w", ownerRepo, fileName, err)
	}

	branch := ref
	if commit.SHA == ref || strings.HasPrefix(commit.SHA, ref) {
		branch = meta.DefaultBranch
	}

	short := commit.SHA
	if len(short) > 7 {
		short = short[:7]
	}
	return &Resolved{
		Repo:     ownerRepo,
		Ref:      ref,
		Branch:   branch,
		Manifest: m,
		Git: &git.Info{
			Remote:   ownerRepo,
			Branch:   branch,
			FullSHA:  commit.SHA,
			ShortSHA: short,
		},
	}, nil
}

type repoInfo struct {
	DefaultBranch string `json:"default_branch"`
}

type commitInfo struct {
	SHA string `json:"sha"`
}

type contentFile struct {
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
	Message  string `json:"message"`
}

func (c *Client) repo(ownerRepo string) (*repoInfo, error) {
	var info repoInfo
	if err := c.get("/repos/"+ownerRepo, &info); err != nil {
		return nil, fmt.Errorf("github.com/%s: %w", ownerRepo, err)
	}
	return &info, nil
}

func (c *Client) commit(ownerRepo, ref string) (*commitInfo, error) {
	var info commitInfo
	path := "/repos/" + ownerRepo + "/commits/" + url.PathEscape(ref)
	if err := c.get(path, &info); err != nil {
		return nil, fmt.Errorf("github.com/%s@%s: %w", ownerRepo, ref, err)
	}
	if info.SHA == "" {
		return nil, fmt.Errorf("github.com/%s@%s: empty commit sha", ownerRepo, ref)
	}
	return &info, nil
}

func (c *Client) file(ownerRepo, sha, path string) ([]byte, error) {
	var cf contentFile
	q := url.Values{"ref": {sha}}
	apiPath := "/repos/" + ownerRepo + "/contents/" + path + "?" + q.Encode()
	if err := c.get(apiPath, &cf); err != nil {
		return nil, fmt.Errorf("github.com/%s %s: %w (add the file to the app repo, or set github.token for a private repo)", ownerRepo, path, err)
	}
	raw := strings.ReplaceAll(cf.Content, "\n", "")
	switch strings.ToLower(cf.Encoding) {
	case "", "base64":
		data, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", path, err)
		}
		return data, nil
	default:
		return nil, fmt.Errorf("unsupported content encoding %q for %s", cf.Encoding, path)
	}
}

func (c *Client) get(path string, dest any) error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		var er struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &er) == nil && er.Message != "" {
			msg = er.Message
		}
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("%s", msg)
		}
		return fmt.Errorf("%s: %s", resp.Status, msg)
	}
	if dest == nil {
		return nil
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("decode github response: %w", err)
	}
	return nil
}
