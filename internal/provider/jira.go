package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/radu103/git-enterprise-hooks/internal/config"
	"github.com/radu103/git-enterprise-hooks/internal/domain"
	"github.com/radu103/git-enterprise-hooks/internal/errs"
)

type JiraClient struct{}

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type jiraSearchResponse struct {
	Issues []struct {
		Key    string `json:"key"`
		Fields struct {
			Summary string `json:"summary"`
			Status  struct {
				Name string `json:"name"`
			} `json:"status"`
			Parent *struct {
				Key string `json:"key"`
			} `json:"parent"`
		} `json:"fields"`
	} `json:"issues"`
}

func (j *JiraClient) Authenticate(ctx context.Context, cfg config.ProviderConfig, username, password string) (domain.AuthToken, error) {
	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("username", username)
	form.Set("password", password)

	var tok oauthTokenResponse
	if err := postForm(ctx, cfg.TokenURL, form, &tok); err != nil {
		return domain.AuthToken{}, err
	}
	return domain.AuthToken{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		TokenType:    tok.TokenType,
		ExpiresAt:    time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second),
		ProviderURL:  cfg.TokenURL,
	}, nil
}

func (j *JiraClient) RefreshToken(ctx context.Context, cfg config.ProviderConfig, tok domain.AuthToken) (domain.AuthToken, error) {
	if strings.TrimSpace(tok.RefreshToken) == "" {
		return domain.AuthToken{}, fmt.Errorf(errs.RefreshTokenMissing)
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("refresh_token", tok.RefreshToken)

	var out oauthTokenResponse
	if err := postForm(ctx, cfg.TokenURL, form, &out); err != nil {
		return domain.AuthToken{}, err
	}
	return domain.AuthToken{
		AccessToken:  out.AccessToken,
		RefreshToken: firstNonEmpty(out.RefreshToken, tok.RefreshToken),
		TokenType:    out.TokenType,
		ExpiresAt:    time.Now().Add(time.Duration(out.ExpiresIn) * time.Second),
		ProviderURL:  cfg.TokenURL,
	}, nil
}

func (j *JiraClient) SearchTasks(ctx context.Context, cfg config.ProviderConfig, tok domain.AuthToken, query string) ([]domain.Task, error) {
	base := strings.TrimSuffix(cfg.TokenURL, "/")
	if strings.Contains(base, "/oauth") {
		base = strings.Split(base, "/oauth")[0]
	}
	jql := fmt.Sprintf("project=%s AND (summary~\"%s\" OR key~\"%s\") ORDER BY updated DESC", cfg.ProjectKey, escapeJQL(query), escapeJQL(query))

	u, err := url.Parse(base + "/rest/api/3/search")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("maxResults", "20")
	q.Set("jql", jql)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	authHeader := "Bearer " + tok.AccessToken
	if strings.TrimSpace(tok.TokenType) != "" {
		authHeader = tok.TokenType + " " + tok.AccessToken
	}
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(errs.FmtJiraSearchFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 410 {
		// New Jira Cloud requires POST /rest/api/3/search/jql with JSON body
		jqlBody := map[string]any{"jql": jql, "maxResults": 20}
		b, _ := json.Marshal(jqlBody)
		u2 := base + "/rest/api/3/search/jql"
		req2, err2 := http.NewRequestWithContext(ctx, http.MethodPost, u2, bytes.NewBuffer(b))
		if err2 == nil {
			req2.Header.Set("Authorization", req.Header.Get("Authorization"))
			req2.Header.Set("Accept", "application/json")
			req2.Header.Set("Content-Type", "application/json")
			resp2, err2 := http.DefaultClient.Do(req2)
			if err2 == nil {
				defer resp2.Body.Close()
				if resp2.StatusCode < 300 {
					var body2 jiraSearchResponse
					if err := json.NewDecoder(resp2.Body).Decode(&body2); err == nil {
						out := make([]domain.Task, 0, len(body2.Issues))
						for _, issue := range body2.Issues {
							epic := ""
							if issue.Fields.Parent != nil {
								epic = issue.Fields.Parent.Key
							}
							out = append(out, domain.Task{
								Key:      issue.Key,
								Title:    issue.Fields.Summary,
								Epic:     epic,
								Status:   issue.Fields.Status.Name,
								Provider: "jira",
							})
						}
						return out, nil
					}
				}
			}
		}
	}

	if resp.StatusCode >= 300 {
		// include response body for easier debugging
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s: %s", fmt.Sprintf(errs.FmtJiraSearchFailedWithStatus, resp.Status), strings.TrimSpace(string(bodyBytes)))
	}

	var body jiraSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	out := make([]domain.Task, 0, len(body.Issues))
	for _, issue := range body.Issues {
		epic := ""
		if issue.Fields.Parent != nil {
			epic = issue.Fields.Parent.Key
		}
		out = append(out, domain.Task{
			Key:      issue.Key,
			Title:    issue.Fields.Summary,
			Epic:     epic,
			Status:   issue.Fields.Status.Name,
			Provider: "jira",
		})
	}
	return out, nil
}

func (j *JiraClient) IsClosedTask(task domain.Task) bool {
	s := strings.ToLower(task.Status)
	return strings.Contains(s, "done") || strings.Contains(s, "closed") || strings.Contains(s, "resolved")
}

func postForm(ctx context.Context, endpoint string, form url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf(errs.FmtOAuthRequestFailed, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf(errs.FmtOAuthRequestFailedStatus, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func firstNonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func escapeJQL(v string) string {
	return strings.ReplaceAll(v, "\"", "")
}
