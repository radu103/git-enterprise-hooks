package provider

import (
	"context"
	"strings"

	"github.com/radu103/git-enterprise-hooks/internal/config"
	"github.com/radu103/git-enterprise-hooks/internal/domain"
)

type Client interface {
	Authenticate(ctx context.Context, cfg config.ProviderConfig, username, password string) (domain.AuthToken, error)
	RefreshToken(ctx context.Context, cfg config.ProviderConfig, tok domain.AuthToken) (domain.AuthToken, error)
	SearchTasks(ctx context.Context, cfg config.ProviderConfig, tok domain.AuthToken, query string) ([]domain.Task, error)
	IsClosedTask(task domain.Task) bool
}

func New(providerType string) (Client, error) {
	switch strings.ToLower(strings.TrimSpace(providerType)) {
	case "jira":
		return &JiraClient{}, nil
	case "github", "github_issues", "github-issues":
		return &GithubClient{}, nil
	default:
		return nil, ErrUnsupportedProvider(providerType)
	}
}
