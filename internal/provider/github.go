package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/radu103/git-enterprise-hooks/internal/config"
	"github.com/radu103/git-enterprise-hooks/internal/domain"
	"github.com/radu103/git-enterprise-hooks/internal/errs"
)

type GithubClient struct{}

type githubSearchResponse struct {
	Items []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		State  string `json:"state"`
	} `json:"items"`
}

type githubIssue struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	State       string `json:"state"`
	Body        string `json:"body"`
	ParentIssue struct {
		Number int `json:"number"`
	} `json:"parent_issue,omitempty"`
	ParentIssueURL string `json:"parent_issue_url,omitempty"`
	PullRequest    *struct {
		URL string `json:"url"`
	} `json:"pull_request,omitempty"`
}

func (g *GithubClient) Authenticate(ctx context.Context, cfg config.ProviderConfig, username, password string) (domain.AuthToken, error) {
	return domain.AuthToken{}, fmt.Errorf(errs.GithubAuthenticationPATBased)
}

func (g *GithubClient) RefreshToken(ctx context.Context, cfg config.ProviderConfig, tok domain.AuthToken) (domain.AuthToken, error) {
	// GitHub PATs are managed outside this tool and do not support refresh.
	if strings.TrimSpace(tok.AccessToken) == "" {
		return domain.AuthToken{}, fmt.Errorf(errs.GithubTokenIsEmpty)
	}
	return tok, nil
}

func (g *GithubClient) SearchTasks(ctx context.Context, cfg config.ProviderConfig, tok domain.AuthToken, query string) ([]domain.Task, error) {
	if strings.TrimSpace(tok.AccessToken) == "" {
		return nil, fmt.Errorf(errs.GithubTokenIsEmpty)
	}
	if strings.TrimSpace(cfg.ProjectKey) == "" {
		return nil, fmt.Errorf(errs.GithubProjectMissing)
	}
	owner, repo, err := splitRepoKey(cfg.ProjectKey)
	if err != nil {
		return nil, err
	}

	apiBase := githubAPIBase(cfg.TokenURL)
	trimmedQuery := strings.TrimSpace(query)

	if issueNumber, ok := parseIssueNumber(trimmedQuery); ok {
		issue, err := g.getIssueByNumber(ctx, apiBase, owner, repo, issueNumber, tok.AccessToken)
		if err != nil {
			return nil, err
		}
		if issue.PullRequest != nil {
			return []domain.Task{}, nil
		}
		return []domain.Task{toDomainTask(issue)}, nil
	}

	if trimmedQuery == "" {
		issues, err := g.listRepoIssues(ctx, apiBase, owner, repo, tok.AccessToken)
		if err != nil {
			return nil, err
		}
		return toDomainTasks(issues), nil
	}

	q := []string{"repo:" + cfg.ProjectKey, "is:issue"}
	q = append(q, trimmedQuery)

	u, err := url.Parse(apiBase + "/search/issues")
	if err != nil {
		return nil, err
	}
	vals := u.Query()
	vals.Set("q", strings.Join(q, " "))
	vals.Set("per_page", "100")
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
		return nil, fmt.Errorf(errs.FmtGithubSearchFailed, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf(errs.FmtGithubSearchFailedWithStatus, resp.Status)
	}

	var body githubSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	out := make([]domain.Task, 0, len(body.Items))
	for _, item := range body.Items {
		issue := githubIssue{
			Number: item.Number,
			Title:  item.Title,
			State:  item.State,
		}
		details, detailsErr := g.getIssueByNumber(ctx, apiBase, owner, repo, item.Number, tok.AccessToken)
		if detailsErr == nil {
			issue = details
		}
		out = append(out, toDomainTask(issue))
	}
	return out, nil
}

func (g *GithubClient) listRepoIssues(ctx context.Context, apiBase, owner, repo, token string) ([]githubIssue, error) {
	u, err := url.Parse(apiBase + "/repos/" + owner + "/" + repo + "/issues")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("state", "all")
	q.Set("sort", "updated")
	q.Set("direction", "desc")
	q.Set("per_page", "100")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(errs.FmtGithubIssueListFailed, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf(errs.FmtGithubIssueListFailedStatus, resp.Status)
	}

	var body []githubIssue
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	return body, nil
}

func (g *GithubClient) getIssueByNumber(ctx context.Context, apiBase, owner, repo string, number int, token string) (githubIssue, error) {
	u := fmt.Sprintf("%s/repos/%s/%s/issues/%d", apiBase, owner, repo, number)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return githubIssue{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return githubIssue{}, fmt.Errorf(errs.FmtGithubIssueLookupFailed, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return githubIssue{}, fmt.Errorf(errs.FmtGithubIssueLookupFailedStatus, resp.Status)
	}

	var issue githubIssue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return githubIssue{}, err
	}
	return issue, nil
}

func splitRepoKey(projectKey string) (string, string, error) {
	parts := strings.Split(strings.TrimSpace(projectKey), "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf(errs.FmtGithubProjectInvalid, projectKey)
	}
	return parts[0], parts[1], nil
}

func parseIssueNumber(query string) (int, bool) {
	if query == "" {
		return 0, false
	}
	q := strings.TrimSpace(strings.TrimPrefix(query, "#"))
	n, err := strconv.Atoi(q)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func toDomainTask(issue githubIssue) domain.Task {
	epic := parentIssueKey(issue)
	return domain.Task{
		Key:      "#" + strconv.Itoa(issue.Number),
		Title:    issue.Title,
		Epic:     epic,
		Status:   issue.State,
		Provider: "github",
	}
}

func toDomainTasks(issues []githubIssue) []domain.Task {
	out := make([]domain.Task, 0, len(issues))
	for _, issue := range issues {
		if issue.PullRequest != nil {
			continue
		}
		out = append(out, toDomainTask(issue))
	}
	return out
}

func parentIssueKey(issue githubIssue) string {
	if issue.ParentIssue.Number > 0 {
		return "#" + strconv.Itoa(issue.ParentIssue.Number)
	}
	if n := parseIssueNumberFromURL(issue.ParentIssueURL); n > 0 {
		return "#" + strconv.Itoa(n)
	}
	if n := parseParentFromBody(issue.Body); n > 0 {
		return "#" + strconv.Itoa(n)
	}
	return ""
}

func parseIssueNumberFromURL(raw string) int {
	if strings.TrimSpace(raw) == "" {
		return 0
	}
	parts := strings.Split(strings.TrimSuffix(strings.TrimSpace(raw), "/"), "/")
	if len(parts) == 0 {
		return 0
	}
	n, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func parseParentFromBody(body string) int {
	if strings.TrimSpace(body) == "" {
		return 0
	}
	re := regexp.MustCompile(`(?i)(?:parent\s*[:#-]?\s*#|epic\s*[:#-]?\s*#)(\d+)`)
	m := re.FindStringSubmatch(body)
	if len(m) < 2 {
		return 0
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n <= 0 {
		return 0
	}
	return n
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
