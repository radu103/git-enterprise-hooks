package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/radu103/git-enterprise-hooks/internal/commit"
	"github.com/radu103/git-enterprise-hooks/internal/config"
	"github.com/radu103/git-enterprise-hooks/internal/domain"
	"github.com/radu103/git-enterprise-hooks/internal/errs"
	"github.com/radu103/git-enterprise-hooks/internal/gitutil"
	"github.com/radu103/git-enterprise-hooks/internal/provider"
	"github.com/radu103/git-enterprise-hooks/internal/ui"
	"golang.org/x/term"
)

func RunPreCommit(ctx context.Context, repoRoot string, cfg config.Config, cfgPath string) error {
	if shouldSkipValidation() {
		fmt.Println("git-enterprise-hooks: non-interactive commit mode detected; using fallback commit message.")
		return writeFallbackCommitMessage(repoRoot, cfg)
	}

	if !providerConfigComplete(cfg) {
		if !isInteractiveSession() {
			return fmt.Errorf(errs.ProviderSetupIncompleteNonInteractive)
		}
		if err := SetupProviderConfig(&cfg); err != nil {
			return err
		}
		if err := config.Save(cfgPath, cfg); err != nil {
			return err
		}
	}

	applyProviderEnvFallback(&cfg)

	cli, err := provider.New(strings.ToLower(cfg.Rules.Provider.Type))
	if err != nil {
		return err
	}

	tok, err := authTokenFromConfig(cfg)
	if err != nil {
		return err
	}

	interactive := isInteractiveSession()
	query := ""
	if interactive {
		query, err = ui.Ask("Search task by key or title", "")
		if err != nil {
			return err
		}
	} else {
		branch, branchErr := gitutil.CurrentBranch(repoRoot)
		if branchErr == nil {
			query = extractTaskKeyFromBranch(cfg.Rules.VerifyBranchName, branch)
		}
		if strings.TrimSpace(query) != "" {
			fmt.Printf("Non-interactive mode: using task key inferred from branch: %s\n", query)
		}
	}
	tasks, err := cli.SearchTasks(ctx, cfg.Rules.Provider, tok, query)
	if err != nil {
		fmt.Printf("Task lookup failed for provider '%s': %v\n", cfg.Rules.Provider.Type, err)
		fmt.Println("Continuing with placeholder values: task_key=NONE, task_title=NONE, task_epic=NONE.")
		tasks = nil
	}
	if cfg.Rules.RejectClosed {
		tasks = filterOpenTasks(cli, tasks)
	}

	skipBranchValidation := false
	var selected *domain.Task

	if len(tasks) == 0 {
		skipBranchValidation = true
		selected = &domain.Task{
			Key:      "NONE",
			Title:    "NONE",
			Epic:     "NONE",
			Status:   "NONE",
			Provider: cfg.Rules.Provider.Type,
		}
		fmt.Println("No tasks found. Continuing with placeholder values: task_key=NONE, task_title=NONE, task_epic=NONE.")
	} else {
		selected, err = ui.SelectTask(tasks)
		if err != nil {
			return err
		}
		if selected == nil {
			return fmt.Errorf(errs.TaskSelectionCanceled)
		}
	}

	if !skipBranchValidation && strings.TrimSpace(cfg.Rules.VerifyBranchName) != "" {
		branch, err := gitutil.CurrentBranch(repoRoot)
		if err != nil {
			return err
		}
		branchTaskKey := normalizeTaskKeyForBranch(selected.Key)
		if err := validateBranch(cfg.Rules.VerifyBranchName, branch, branchTaskKey); err != nil {
			return err
		}
	}

	editedTitle, err := ui.Ask("Task title", selected.Title)
	if err != nil {
		return err
	}
	summary, err := ui.Ask("Commit summary", "")
	if err != nil {
		return err
	}
	msg := commit.Render(cfg.Message, domain.CommitMessageContext{
		TaskKey:   selected.Key,
		TaskTitle: editedTitle,
		TaskEpic:  selected.Epic,
		Summary:   summary,
	})
	if strings.TrimSpace(msg) == "" {
		return fmt.Errorf(errs.FormattedCommitMessageEmpty)
	}

	outPath := filepath.Join(repoRoot, ".git", "git-enterprise-hooks-message.txt")
	if err := os.WriteFile(outPath, []byte(msg), 0o644); err != nil {
		return err
	}
	commitMsgPath, err := gitutil.GitPath(repoRoot, "COMMIT_EDITMSG")
	if err == nil {
		_ = os.WriteFile(commitMsgPath, []byte(msg), 0o644)
	}
	fmt.Println("--- git-enterprise-hooks commit message ---")
	fmt.Println(msg)
	fmt.Printf("\nSaved to %s\n", outPath)
	if err == nil {
		fmt.Printf("Prepared commit message editor file: %s\n", commitMsgPath)
	}
	return nil
}

func writeFallbackCommitMessage(repoRoot string, cfg config.Config) error {
	fallbackTask, err := nonInteractiveFallbackTask(repoRoot, cfg, "")
	if err != nil {
		return err
	}

	msg := commit.Render(cfg.Message, domain.CommitMessageContext{
		TaskKey:   fallbackTask.Key,
		TaskTitle: fallbackTask.Title,
		TaskEpic:  fallbackTask.Epic,
		Summary:   "",
	})
	if strings.TrimSpace(msg) == "" {
		return fmt.Errorf(errs.FormattedCommitMessageEmpty)
	}

	outPath := filepath.Join(repoRoot, ".git", "git-enterprise-hooks-message.txt")
	if err := os.WriteFile(outPath, []byte(msg), 0o644); err != nil {
		return err
	}

	commitMsgPath, err := gitutil.GitPath(repoRoot, "COMMIT_EDITMSG")
	if err == nil {
		_ = os.WriteFile(commitMsgPath, []byte(msg), 0o644)
	}

	return nil
}

func SetupProviderConfig(cfg *config.Config) error {
	return setupProviderConfig(cfg)
}

func setupProviderConfig(cfg *config.Config) error {
	providerType, err := ui.Ask("Provider type (jira/azure_devops/github)", cfg.Rules.Provider.Type)
	if err != nil {
		return err
	}
	cfg.Rules.Provider.Type = strings.ToLower(strings.TrimSpace(providerType))

	switch cfg.Rules.Provider.Type {
	case "github":
		if strings.TrimSpace(githubURL(*cfg)) == "" {
			cfg.ProviderGithub.URL, err = ui.Ask("GitHub URL", cfg.ProviderGithub.URL)
			if err != nil {
				return err
			}
		}
		if strings.TrimSpace(githubProject(*cfg)) == "" {
			cfg.ProviderGithub.Project, err = ui.Ask("GitHub project", cfg.ProviderGithub.Project)
			if err != nil {
				return err
			}
		}
		if strings.TrimSpace(githubToken(*cfg)) == "" {
			cfg.ProviderGithub.GithubPAT, err = ui.AskPassword("GitHub PAT (leave empty to use GITHUB_TOKEN or GITHUB_PAT)")
			if err != nil {
				return err
			}
		}
		cfg.ProviderJira = config.JiraProviderConfig{}
		cfg.ProviderAzureDevOps = config.AzureDevOpsProviderConfig{}
	case "azure_devops", "azure-devops", "devops":
		cfg.Rules.Provider.Type = "azure_devops"
		if strings.TrimSpace(azureDevOpsURL(*cfg)) == "" {
			cfg.ProviderAzureDevOps.URL, err = ui.Ask("Azure DevOps URL", cfg.ProviderAzureDevOps.URL)
			if err != nil {
				return err
			}
		}
		if strings.TrimSpace(azureDevOpsProject(*cfg)) == "" {
			cfg.ProviderAzureDevOps.Project, err = ui.Ask("Azure DevOps project", cfg.ProviderAzureDevOps.Project)
			if err != nil {
				return err
			}
		}
		if strings.TrimSpace(azureDevOpsToken(*cfg)) == "" {
			cfg.ProviderAzureDevOps.PersonalAccessToken, err = ui.AskPassword("Azure DevOps Personal Access Token (leave empty to use AZURE_DEVOPS_PAT or AZURE_DEVOPS_EXT_PAT)")
			if err != nil {
				return err
			}
		}
		cfg.ProviderJira = config.JiraProviderConfig{}
		cfg.ProviderGithub = config.GithubProviderConfig{}
	default:
		cfg.Rules.Provider.Type = "jira"
		if strings.TrimSpace(jiraURL(*cfg)) == "" {
			cfg.ProviderJira.URL, err = ui.Ask("Jira URL", cfg.ProviderJira.URL)
			if err != nil {
				return err
			}
		}
		if strings.TrimSpace(jiraProject(*cfg)) == "" {
			cfg.ProviderJira.Project, err = ui.Ask("Jira project", cfg.ProviderJira.Project)
			if err != nil {
				return err
			}
		}
		if strings.TrimSpace(jiraUserEmail(*cfg)) == "" {
			cfg.ProviderJira.UserEmail, err = ui.Ask("Jira user email", cfg.ProviderJira.UserEmail)
			if err != nil {
				return err
			}
		}
		if strings.TrimSpace(jiraToken(*cfg)) == "" {
			cfg.ProviderJira.JiraAPIToken, err = ui.AskPassword("Jira API token (leave empty to use JIRA_API_TOKEN or JIRA_TOKEN)")
			if err != nil {
				return err
			}
		}
		cfg.ProviderGithub = config.GithubProviderConfig{}
		cfg.ProviderAzureDevOps = config.AzureDevOpsProviderConfig{}
	}

	return nil
}

func providerConfigComplete(cfg config.Config) bool {
	switch strings.ToLower(strings.TrimSpace(cfg.Rules.Provider.Type)) {
	case "github":
		return strings.TrimSpace(githubURL(cfg)) != "" &&
			strings.TrimSpace(githubProject(cfg)) != "" &&
			strings.TrimSpace(githubToken(cfg)) != ""
	case "azure_devops", "azure-devops", "devops":
		return strings.TrimSpace(azureDevOpsURL(cfg)) != "" &&
			strings.TrimSpace(azureDevOpsProject(cfg)) != "" &&
			strings.TrimSpace(azureDevOpsToken(cfg)) != ""
	default:
		return strings.TrimSpace(jiraURL(cfg)) != "" &&
			strings.TrimSpace(jiraProject(cfg)) != "" &&
			strings.TrimSpace(jiraUserEmail(cfg)) != "" &&
			strings.TrimSpace(jiraToken(cfg)) != ""
	}
}

func authTokenFromConfig(cfg config.Config) (domain.AuthToken, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Rules.Provider.Type)) {
	case "github":
		pat := githubToken(cfg)
		if strings.TrimSpace(pat) == "" {
			return domain.AuthToken{}, fmt.Errorf(errs.MissingGithubToken)
		}
		return domain.AuthToken{AccessToken: pat, TokenType: "Bearer"}, nil
	case "azure_devops", "azure-devops", "devops":
		pat := azureDevOpsToken(cfg)
		if strings.TrimSpace(pat) == "" {
			return domain.AuthToken{}, fmt.Errorf(errs.MissingAzureDevOpsToken)
		}
		return domain.AuthToken{AccessToken: pat, TokenType: "Bearer"}, nil
	default:
		tok := jiraToken(cfg)
		if strings.TrimSpace(tok) == "" {
			return domain.AuthToken{}, fmt.Errorf(errs.MissingJiraToken)
		}
		return domain.AuthToken{AccessToken: tok, TokenType: "Bearer"}, nil
	}
}

func applyProviderEnvFallback(cfg *config.Config) {
	switch strings.ToLower(strings.TrimSpace(cfg.Rules.Provider.Type)) {
	case "github":
		cfg.ProviderGithub.URL = githubURL(*cfg)
		cfg.ProviderGithub.Project = githubProject(*cfg)
		cfg.Rules.Provider.TokenURL = cfg.ProviderGithub.URL
		cfg.Rules.Provider.ProjectKey = cfg.ProviderGithub.Project
	case "azure_devops", "azure-devops", "devops":
		cfg.Rules.Provider.Type = "azure_devops"
		cfg.ProviderAzureDevOps.URL = azureDevOpsURL(*cfg)
		cfg.ProviderAzureDevOps.Project = azureDevOpsProject(*cfg)
		cfg.Rules.Provider.TokenURL = cfg.ProviderAzureDevOps.URL
		cfg.Rules.Provider.ProjectKey = cfg.ProviderAzureDevOps.Project
	default:
		cfg.Rules.Provider.Type = "jira"
		cfg.ProviderJira.URL = jiraURL(*cfg)
		cfg.ProviderJira.Project = jiraProject(*cfg)
		cfg.ProviderJira.UserEmail = jiraUserEmail(*cfg)
		cfg.Rules.Provider.TokenURL = cfg.ProviderJira.URL
		cfg.Rules.Provider.ProjectKey = cfg.ProviderJira.Project
	}
}

func githubURL(cfg config.Config) string {
	return firstNonEmpty(cfg.ProviderGithub.URL, os.Getenv("GITHUB_URL"))
}

func githubProject(cfg config.Config) string {
	return firstNonEmpty(cfg.ProviderGithub.Project, os.Getenv("GITHUB_PROJECT"))
}

func githubToken(cfg config.Config) string {
	return firstNonEmpty(cfg.ProviderGithub.GithubPAT, os.Getenv("GITHUB_PAT"), os.Getenv("GITHUB_TOKEN"))
}

func jiraURL(cfg config.Config) string {
	return firstNonEmpty(cfg.ProviderJira.URL, os.Getenv("JIRA_URL"))
}

func jiraProject(cfg config.Config) string {
	return firstNonEmpty(cfg.ProviderJira.Project, os.Getenv("JIRA_PROJECT"))
}

func jiraUserEmail(cfg config.Config) string {
	return firstNonEmpty(cfg.ProviderJira.UserEmail, os.Getenv("JIRA_USER_EMAIL"))
}

func jiraToken(cfg config.Config) string {
	return firstNonEmpty(cfg.ProviderJira.JiraAPIToken, os.Getenv("JIRA_API_TOKEN"), os.Getenv("JIRA_TOKEN"))
}

func azureDevOpsURL(cfg config.Config) string {
	return firstNonEmpty(cfg.ProviderAzureDevOps.URL, os.Getenv("AZURE_DEVOPS_URL"), os.Getenv("AZDO_URL"))
}

func azureDevOpsProject(cfg config.Config) string {
	return firstNonEmpty(cfg.ProviderAzureDevOps.Project, os.Getenv("AZURE_DEVOPS_PROJECT"), os.Getenv("AZDO_PROJECT"))
}

func azureDevOpsToken(cfg config.Config) string {
	return firstNonEmpty(cfg.ProviderAzureDevOps.PersonalAccessToken, os.Getenv("AZURE_DEVOPS_PAT"), os.Getenv("AZURE_DEVOPS_EXT_PAT"), os.Getenv("SYSTEM_ACCESSTOKEN"))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func filterOpenTasks(cli provider.Client, tasks []domain.Task) []domain.Task {
	out := make([]domain.Task, 0, len(tasks))
	for _, t := range tasks {
		if cli.IsClosedTask(t) {
			continue
		}
		out = append(out, t)
	}
	return out
}

func normalizeTaskKeyForBranch(taskKey string) string {
	return strings.TrimPrefix(strings.TrimSpace(taskKey), "#")
}

func validateBranch(pattern, branch, taskKey string) error {
	raw := strings.ReplaceAll(pattern, "{task_key}", taskKey)
	regex := regexp.QuoteMeta(raw)
	regex = strings.ReplaceAll(regex, "\\*", ".*")
	regex = "^" + regex + "$"
	ok, err := regexp.MatchString(regex, branch)
	if err != nil {
		return fmt.Errorf(errs.FmtInvalidBranchPattern, err)
	}
	if !ok {
		return fmt.Errorf(errs.FmtInvalidBranchName, branch, raw)
	}
	return nil
}

func isInteractiveSession() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func shouldSkipValidation() bool {
	// Skip only for VS Code SCM non-interactive commits, not for integrated terminal runs.
	if strings.TrimSpace(os.Getenv("VSCODE_GIT_IPC_HANDLE")) != "" &&
		!strings.EqualFold(strings.TrimSpace(os.Getenv("TERM_PROGRAM")), "vscode") &&
		!isInteractiveSession() {
		return true
	}

	return false
}

func nonInteractiveFallbackTask(repoRoot string, cfg config.Config, query string) (domain.Task, error) {
	key := strings.TrimSpace(query)
	if key == "" {
		branch, err := gitutil.CurrentBranch(repoRoot)
		if err != nil {
			return domain.Task{}, err
		}
		key = extractTaskKeyFromBranch(cfg.Rules.VerifyBranchName, branch)
	}
	if strings.TrimSpace(key) == "" {
		key = "NO-TASK"
	}

	return domain.Task{
		Key:      key,
		Title:    "Task inferred for non-interactive commit",
		Epic:     "",
		Status:   "open",
		Provider: cfg.Rules.Provider.Type,
	}, nil
}

func extractTaskKeyFromBranch(pattern, branch string) string {
	if strings.TrimSpace(pattern) == "" || !strings.Contains(pattern, "{task_key}") {
		return ""
	}
	parts := strings.SplitN(pattern, "{task_key}", 2)
	prefix := parts[0]
	suffix := parts[1]

	if !strings.HasPrefix(branch, prefix) {
		return ""
	}
	remaining := strings.TrimPrefix(branch, prefix)

	if suffix == "" {
		return remaining
	}

	suffixParts := strings.SplitN(suffix, "*", 2)
	suffixStatic := suffixParts[0]
	if suffixStatic == "" {
		return remaining
	}
	idx := strings.Index(remaining, suffixStatic)
	if idx <= 0 {
		return ""
	}
	return remaining[:idx]
}
