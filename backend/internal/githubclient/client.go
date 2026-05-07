package githubclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

var (
	ErrInvalidRepositoryURL = errors.New("invalid repository url")
	ErrProtoNotFound        = errors.New("project.proto not found")
)

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	httpClient httpDoer
	token      string
}

func New(httpClient httpDoer, token string) *Client {
	return &Client{httpClient: httpClient, token: token}
}

func (c *Client) ReadProjectProto(ctx context.Context, repositoryURL, ref string) ([]byte, error) {
	owner, repo, err := parseOwnerRepo(repositoryURL)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/project.proto", owner, repo)
	if strings.TrimSpace(ref) != "" {
		apiURL += "?ref=" + url.QueryEscape(ref)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if strings.TrimSpace(c.token) != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrProtoNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github api status: %d", resp.StatusCode)
	}

	var body struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if body.Encoding != "base64" {
		return nil, fmt.Errorf("unsupported github content encoding: %s", body.Encoding)
	}

	raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(body.Content, "\n", ""))
	if err != nil {
		return nil, err
	}

	return raw, nil
}

func parseOwnerRepo(repositoryURL string) (string, string, error) {
	u, err := url.Parse(strings.TrimSpace(repositoryURL))
	if err != nil || u.Host == "" {
		return "", "", ErrInvalidRepositoryURL
	}
	if !strings.EqualFold(u.Host, "github.com") {
		return "", "", ErrInvalidRepositoryURL
	}

	path := strings.Trim(strings.TrimSuffix(u.Path, ".git"), "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", ErrInvalidRepositoryURL
	}

	return parts[0], parts[1], nil
}

