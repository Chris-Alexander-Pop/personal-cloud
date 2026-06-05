package woodpecker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

type Repo struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Owner    string `json:"owner"`
	FullName string `json:"full_name"`
}

type Pipeline struct {
	ID     int64  `json:"id"`
	Number int64  `json:"number"`
	State  string `json:"state"`
	Error  string `json:"error"`
}

type PipelineOptions struct {
	Branch    string            `json:"branch"`
	Variables map[string]string `json:"variables"`
}

type repoCache struct {
	Repos map[string]int64 `json:"repos"`
}

func New(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) CreatePipeline(repoID int64, opts PipelineOptions) (*Pipeline, error) {
	body, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/repos/%d/pipelines", c.baseURL, repoID), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create pipeline: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	var pl Pipeline
	if err := json.NewDecoder(resp.Body).Decode(&pl); err != nil {
		return nil, err
	}
	return &pl, nil
}

func (c *Client) GetPipeline(repoID, number int64) (*Pipeline, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/repos/%d/pipelines/%d", c.baseURL, repoID, number), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get pipeline: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	var pl Pipeline
	if err := json.NewDecoder(resp.Body).Decode(&pl); err != nil {
		return nil, err
	}
	return &pl, nil
}

func (c *Client) RepoID(owner, name, cachePath string) (int64, error) {
	key := owner + "/" + name
	if cachePath != "" {
		if id, ok := loadCache(cachePath, key); ok {
			return id, nil
		}
	}
	repos, err := c.listRepos()
	if err != nil {
		return 0, err
	}
	cache := repoCache{Repos: make(map[string]int64)}
	for _, r := range repos {
		slug := r.Owner + "/" + r.Name
		if r.FullName != "" {
			slug = r.FullName
		}
		cache.Repos[slug] = r.ID
	}
	if cachePath != "" {
		_ = saveCache(cachePath, cache)
	}
	id, ok := cache.Repos[key]
	if !ok {
		return 0, fmt.Errorf("repo %s not found in Woodpecker (activate it in the UI)", key)
	}
	return id, nil
}

func (c *Client) listRepos() ([]Repo, error) {
	var all []Repo
	page := 1
	for {
		u := fmt.Sprintf("%s/api/user/repos?all=true&page=%d&per_page=50", c.baseURL, page)
		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("list repos: %s: %s", resp.Status, strings.TrimSpace(string(body)))
		}
		var batch []Repo
		if err := json.Unmarshal(body, &batch); err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		all = append(all, batch...)
		if len(batch) < 50 {
			break
		}
		page++
	}
	return all, nil
}

func (c *Client) Ping() error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/api/user", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("woodpecker ping: %s", resp.Status)
	}
	return nil
}

func (c *Client) PipelineURL(repoID, number int64) string {
	return fmt.Sprintf("%s/repos/%d/%d", c.baseURL, repoID, number)
}

func loadCache(path, key string) (int64, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	var c repoCache
	if json.Unmarshal(data, &c) != nil {
		return 0, false
	}
	id, ok := c.Repos[key]
	return id, ok
}

func saveCache(path string, c repoCache) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Wait polls until pipeline reaches terminal state.
func (c *Client) Wait(repoID, number int64, interval time.Duration) (*Pipeline, error) {
	for {
		pl, err := c.GetPipeline(repoID, number)
		if err != nil {
			return nil, err
		}
		switch strings.ToLower(pl.State) {
		case "success":
			return pl, nil
		case "failure", "error", "killed", "declined":
			msg := pl.Error
			if msg == "" {
				msg = pl.State
			}
			return pl, fmt.Errorf("pipeline failed: %s", msg)
		}
		time.Sleep(interval)
	}
}

// LatestPipelineForRepo finds newest pipeline (best-effort).
func (c *Client) LatestPipelineForRepo(repoID int64) (*Pipeline, error) {
	u := fmt.Sprintf("%s/api/repos/%d/pipelines?page=1&per_page=1", c.baseURL, repoID)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("list pipelines: %s", resp.Status)
	}
	var list []Pipeline
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("no pipelines found")
	}
	return &list[0], nil
}

// NormalizeURL ensures scheme is present.
func NormalizeURL(raw string) string {
	if raw == "" {
		return raw
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		return "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return u.String()
}
