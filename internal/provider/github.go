package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/radu103/git-enterprise-hooks/internal/config"
	"github.com/radu103/git-enterprise-hooks/internal/domain"
)

type GithubClient struct{}

type githubSearchResponse struct {
	Items []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		State  string `json:"state"`
	} `json:"items"`
}

func (g *GithubClient) Authenticate(ctx context.Context, cfg config.ProviderConfig, username, password string) (domain.AuthToken, error) {
	return domain.AuthToken{}, fmt.Errorf("github authentication is PAT-based: set provider_github.github_pat or GITHUB_TOKEN")
}

func (g *GithubClient) RefreshToken(ctx context.Context, cfg config.ProviderConfig, tok domain.AuthToken) (domain.AuthToken, error) {
	// GitHub PATs are managed outside this tool and do not support refresh.
	if strings.TrimSpace(tok.AccessToken) == "" {
		return domain.AuthToken{}, fmt.Errorf("github token is empty")
	}
	return tok, nil
}

func (g *GithubClient) SearchTasks(ctx context.Context, cfg config.ProviderConfig, tok domain.AuthToken, query string) ([]domain.Task, error) {
	if strings.TrimSpace(tok.AccessToken) == "" {
		return nil, fmt.Errorf("github token is empty")
	}
	if strings.TrimSpace(cfg.ProjectKey) == "" {
		return nil, fmt.Errorf("github project is missing (expected owner/repo)")
	}

	apiBase := githubAPIBase(cfg.TokenURL)
	q := []string{"repo:" + cfg.ProjectKey, "is:issue"}
	if strings.TrimSpace(query) != "" {
		q = append(q, query)
	}

	u, err := url.Parse(apiBase + "/search/issues")
	if err != nil {
		return nil, err
	}
	vals := u.Query()
	vals.Set("q", strings.Join(q, " "))
	vals.Set("per_page", "20")
	u.RawQuery = vals.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github search failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github search failed with status %s", resp.Status)
	}

	var body githubSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	out := make([]domain.Task, 0, len(body.Items))
	for _, item := range body.Items {
		out = append(out, domain.Task{
			Key:      "#" + strconv.Itoa(item.Number),
			Title:    item.Title,
			Epic:     "",
			Status:   item.State,
			Provider: "github",
		})
	}
	return out, nil
}

func (g *GithubClient) IsClosedTask(task domain.Task) bool {
	return strings.EqualFold(strings.TrimSpace(task.Status), "closed")
}

func githubAPIBase(baseURL string) string {
	base := strings.TrimSpace(baseURL)
	if base == "" {
		return "https://api.github.com"
	}
	base = strings.TrimSuffix(base, "/")

	// Public GitHub instance.
	if strings.EqualFold(base, "https://github.com") || strings.EqualFold(base, "http://github.com") {
		return "https://api.github.com"
	}

	// GitHub Enterprise API default path.
	if strings.HasSuffix(strings.ToLower(base), "/api/v3") {
		return base
	}
	return base + "/api/v3"
}
