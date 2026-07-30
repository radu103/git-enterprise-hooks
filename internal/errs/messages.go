package errs

const (
	ConfigMessageCannotBeEmpty = "config message cannot be empty"
	ProviderTypeCannotBeEmpty  = "provider type cannot be empty"
	NotInsideGitRepository     = "not inside a git repository"

	ProviderSetupIncompleteNonInteractive = "provider setup is incomplete for non-interactive commit. Run 'git-enterprise-hooks setup' first, or set required provider values in git-enterprise-hooks.yaml/.env"
	TaskSelectionCanceled                 = "task selection canceled"
	FormattedCommitMessageEmpty           = "formatted commit message is empty"
	MissingGithubToken                    = "missing github token: set provider_github.github_pat or GITHUB_TOKEN"
	MissingAzureDevOpsToken               = "missing azure devops token: set provider_azure_devops.personal_access_token or AZURE_DEVOPS_PAT"
	MissingJiraToken                      = "missing jira token: set provider_jira.jira_api_token or JIRA_API_TOKEN"

	GithubAuthenticationPATBased = "github authentication is PAT-based: set provider_github.github_pat or GITHUB_TOKEN"
	GithubTokenIsEmpty           = "github token is empty"
	GithubProjectMissing         = "github project is missing (expected owner/repo)"
	RefreshTokenMissing          = "refresh token missing"

	FmtUnsupportedProviderTypeAllowed = "unsupported provider type '%s' (allowed: jira, azure_devops, github)"
	FmtUnsupportedProviderTypeSimple  = "unsupported provider type: %s"
	FmtDeleteConfigFile               = "delete config file: %w"
	FmtReadConfig                     = "read config: %w"
	FmtParseConfig                    = "parse config: %w"
	FmtCreateConfigFolder             = "create config folder: %w"
	FmtUpdateConfig                   = "update config: %w"
	FmtWriteConfig                    = "write config: %w"

	FmtGetCurrentBranch = "get current branch: %w"
	FmtResolveHooksPath = "resolve hooks path: %w"
	FmtResolveGitPath   = "resolve git path '%s': %w"

	FmtWritePreCommitHook  = "write pre-commit hook: %w"
	FmtRemovePreCommitHook = "remove pre-commit hook: %w"

	FmtGithubSearchFailed            = "github search failed: %w"
	FmtGithubSearchFailedWithStatus  = "github search failed with status %s"
	FmtGithubIssueListFailed         = "github issue list failed: %w"
	FmtGithubIssueListFailedStatus   = "github issue list failed with status %s"
	FmtGithubIssueLookupFailed       = "github issue lookup failed: %w"
	FmtGithubIssueLookupFailedStatus = "github issue lookup failed with status %s"
	FmtGithubProjectInvalid          = "github project is invalid: expected owner/repo, got '%s'"

	FmtJiraSearchFailed           = "jira search failed: %w"
	FmtJiraSearchFailedWithStatus = "jira search failed with status %s"
	FmtOAuthRequestFailed         = "oauth request failed: %w"
	FmtOAuthRequestFailedStatus   = "oauth request failed with status %s"

	FmtInvalidBranchPattern = "invalid branch pattern: %w"
	FmtInvalidBranchName    = "invalid branch name '%s': branch does not follow required naming pattern '%s'. Rename your branch to include a valid task key (example: feature/ABC-123-short-description)"
)
