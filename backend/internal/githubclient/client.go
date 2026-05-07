package githubclient

import (
	"bytes"
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

func (c *Client) ListBranches(ctx context.Context, repositoryURL string) ([]string, error) {
	owner, repo, err := parseOwnerRepo(repositoryURL)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/branches?per_page=100", owner, repo)
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
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github api status: %d", resp.StatusCode)
	}

	var body []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	out := make([]string, 0, len(body))
	for _, b := range body {
		if strings.TrimSpace(b.Name) != "" {
			out = append(out, b.Name)
		}
	}
	return out, nil
}

func (c *Client) ListTags(ctx context.Context, repositoryURL string) ([]string, error) {
	owner, repo, err := parseOwnerRepo(repositoryURL)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/tags?per_page=100", owner, repo)
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
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github api status: %d", resp.StatusCode)
	}

	var body []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	out := make([]string, 0, len(body))
	for _, b := range body {
		if strings.TrimSpace(b.Name) != "" {
			out = append(out, b.Name)
		}
	}
	return out, nil
}

func (c *Client) DispatchWorkflow(ctx context.Context, repositoryURL, workflowID, ref string, inputs map[string]string) error {
	owner, repo, err := parseOwnerRepo(repositoryURL)
	if err != nil {
		return err
	}
	workflowID = strings.TrimSpace(workflowID)
	ref = strings.TrimSpace(ref)
	if workflowID == "" || ref == "" {
		return errors.New("workflow_id and ref are required")
	}

	payload, err := json.Marshal(map[string]any{
		"ref":    ref,
		"inputs": inputs,
	})
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/workflows/%s/dispatches", owner, repo, workflowID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(c.token) != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("github api status: %d", resp.StatusCode)
	}

	return nil
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
