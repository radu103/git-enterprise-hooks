package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/radu103/git-enterprise-hooks/internal/errs"
	"github.com/spf13/viper"
)

const DefaultConfigFile = "git-enterprise-hooks.yaml"

type Config struct {
	Required            bool                      `mapstructure:"required"`
	Format              string                    `mapstructure:"format"`
	Message             string                    `mapstructure:"message"`
	ProviderJira        JiraProviderConfig        `mapstructure:"provider_jira"`
	ProviderGithub      GithubProviderConfig      `mapstructure:"provider_github"`
	ProviderAzureDevOps AzureDevOpsProviderConfig `mapstructure:"provider_azure_devops"`
	Rules               Rules                     `mapstructure:"rules"`
}

type JiraProviderConfig struct {
	URL          string `mapstructure:"url"`
	UserEmail    string `mapstructure:"user_email"`
	JiraAPIToken string `mapstructure:"jira_api_token"`
	Project      string `mapstructure:"project"`
}

type GithubProviderConfig struct {
	URL       string `mapstructure:"url"`
	GithubPAT string `mapstructure:"github_pat"`
	Project   string `mapstructure:"project"`
}

type AzureDevOpsProviderConfig struct {
	URL                 string `mapstructure:"url"`
	PersonalAccessToken string `mapstructure:"personal_access_token"`
	Project             string `mapstructure:"project"`
}

type Rules struct {
	Provider         ProviderConfig `mapstructure:"provider"`
	VerifyBranchName string         `mapstructure:"verify_branch_name"`
	VerifyTaskExists bool           `mapstructure:"verify_task_exists"`
	RejectClosed     bool           `mapstructure:"reject_closed_tasks"`
}

type ProviderConfig struct {
	Type         string `mapstructure:"type"`
	TokenURL     string `mapstructure:"provider_token_url"`
	ClientID     string `mapstructure:"provider_client_id"`
	ClientSecret string `mapstructure:"provider_client_secret"`
	ProjectKey   string `mapstructure:"provider_project_key"`
}

func Default() Config {
	return Config{
		Required: true,
		Format:   "conventional",
		Message:  "[{task_key}] - {task_title}\nEpic: {task_epic}\n\n{summary}",
		Rules: Rules{
			Provider: ProviderConfig{
				Type: "jira",
			},
			VerifyBranchName: "feature/{task_key}-*",
			VerifyTaskExists: true,
			RejectClosed:     true,
		},
	}
}

func ResolvePath(repoRoot, explicit string) string {
	if strings.TrimSpace(explicit) != "" {
		if filepath.IsAbs(explicit) {
			return explicit
		}
		return filepath.Join(repoRoot, explicit)
	}
	return filepath.Join(repoRoot, DefaultConfigFile)
}

func LoadOrCreate(repoRoot, explicitPath string) (Config, string, error) {
	cfgPath := ResolvePath(repoRoot, explicitPath)
	if _, err := os.Stat(cfgPath); errors.Is(err, os.ErrNotExist) {
		cfg := Default()
		if err := Save(cfgPath, cfg); err != nil {
			return Config{}, "", err
		}
		return cfg, cfgPath, nil
	}

	v := viper.New()
	v.SetConfigFile(cfgPath)
	if err := v.ReadInConfig(); err != nil {
		return Config{}, "", fmt.Errorf(errs.FmtReadConfig, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, "", fmt.Errorf(errs.FmtParseConfig, err)
	}
	cfg.normalizeProviders(true)
	if err := cfg.Validate(); err != nil {
		return Config{}, "", err
	}
	return cfg, cfgPath, nil
}

func Save(path string, cfg Config) error {
	cfg.normalizeProviders(false)

	v := viper.New()
	v.SetConfigType("yaml")
	v.Set("required", cfg.Required)
	v.Set("format", cfg.Format)
	v.Set("message", cfg.Message)
	switch cfg.Rules.Provider.Type {
	case "github":
		v.Set("provider_github.url", cfg.ProviderGithub.URL)
		v.Set("provider_github.github_pat", cfg.ProviderGithub.GithubPAT)
		v.Set("provider_github.project", cfg.ProviderGithub.Project)
	case "azure_devops", "azure-devops", "devops":
		v.Set("provider_azure_devops.url", cfg.ProviderAzureDevOps.URL)
		v.Set("provider_azure_devops.personal_access_token", cfg.ProviderAzureDevOps.PersonalAccessToken)
		v.Set("provider_azure_devops.project", cfg.ProviderAzureDevOps.Project)
	default:
		v.Set("provider_jira.url", cfg.ProviderJira.URL)
		v.Set("provider_jira.user_email", cfg.ProviderJira.UserEmail)
		v.Set("provider_jira.jira_api_token", cfg.ProviderJira.JiraAPIToken)
		v.Set("provider_jira.project", cfg.ProviderJira.Project)
	}
	v.Set("rules.verify_branch_name", cfg.Rules.VerifyBranchName)
	v.Set("rules.verify_task_exists", cfg.Rules.VerifyTaskExists)
	v.Set("rules.reject_closed_tasks", cfg.Rules.RejectClosed)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf(errs.FmtCreateConfigFolder, err)
	}
	v.SetConfigFile(path)
	if _, err := os.Stat(path); err == nil {
		if err := v.WriteConfig(); err != nil {
			return fmt.Errorf(errs.FmtUpdateConfig, err)
		}
		return nil
	}
	if err := v.WriteConfigAs(path); err != nil {
		return fmt.Errorf(errs.FmtWriteConfig, err)
	}
	return nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Message) == "" {
		return errors.New(errs.ConfigMessageCannotBeEmpty)
	}
	if strings.TrimSpace(c.Rules.Provider.Type) == "" {
		return errors.New(errs.ProviderTypeCannotBeEmpty)
	}
	return nil
}

func (c *Config) normalizeProviders(loadEnv bool) {
	legacyType := strings.ToLower(strings.TrimSpace(c.Rules.Provider.Type))
	envGithubPAT := ""
	envJiraToken := ""
	envDevOpsPAT := ""
	if loadEnv {
		envGithubPAT = firstNonEmpty(os.Getenv("GITHUB_PAT"), os.Getenv("GITHUB_TOKEN"))
		envJiraToken = firstNonEmpty(os.Getenv("JIRA_API_TOKEN"), os.Getenv("JIRA_TOKEN"))
		envDevOpsPAT = firstNonEmpty(os.Getenv("AZURE_DEVOPS_PAT"), os.Getenv("AZURE_DEVOPS_EXT_PAT"), os.Getenv("SYSTEM_ACCESSTOKEN"))
	}

	if loadEnv {
		if legacyType == "jira" {
			c.ProviderJira.URL = firstNonEmpty(c.ProviderJira.URL, c.Rules.Provider.TokenURL)
			c.ProviderJira.Project = firstNonEmpty(c.ProviderJira.Project, c.Rules.Provider.ProjectKey)
			c.ProviderJira.JiraAPIToken = firstNonEmpty(c.ProviderJira.JiraAPIToken, c.Rules.Provider.ClientSecret)
		}
		if legacyType == "github" {
			c.ProviderGithub.URL = firstNonEmpty(c.ProviderGithub.URL, c.Rules.Provider.TokenURL)
			c.ProviderGithub.Project = firstNonEmpty(c.ProviderGithub.Project, c.Rules.Provider.ProjectKey)
			c.ProviderGithub.GithubPAT = firstNonEmpty(c.ProviderGithub.GithubPAT, c.Rules.Provider.ClientSecret)
		}
		if legacyType == "azure_devops" || legacyType == "azure-devops" || legacyType == "devops" {
			c.ProviderAzureDevOps.URL = firstNonEmpty(c.ProviderAzureDevOps.URL, c.Rules.Provider.TokenURL)
			c.ProviderAzureDevOps.Project = firstNonEmpty(c.ProviderAzureDevOps.Project, c.Rules.Provider.ProjectKey)
			c.ProviderAzureDevOps.PersonalAccessToken = firstNonEmpty(c.ProviderAzureDevOps.PersonalAccessToken, c.Rules.Provider.ClientSecret)
		}
	}

	providerType := legacyType
	if providerType == "" {
		switch {
		case strings.TrimSpace(c.ProviderJira.URL) != "" || strings.TrimSpace(c.ProviderJira.JiraAPIToken) != "":
			providerType = "jira"
		case strings.TrimSpace(c.ProviderGithub.URL) != "" || strings.TrimSpace(c.ProviderGithub.GithubPAT) != "":
			providerType = "github"
		case strings.TrimSpace(c.ProviderAzureDevOps.URL) != "" || strings.TrimSpace(c.ProviderAzureDevOps.PersonalAccessToken) != "":
			providerType = "azure_devops"
		default:
			providerType = "jira"
		}
	}

	if providerType == "azure-devops" || providerType == "devops" {
		providerType = "azure_devops"
	}

	c.Rules.Provider.Type = providerType

	switch providerType {
	case "github":
		c.Rules.Provider.TokenURL = firstNonEmpty(c.Rules.Provider.TokenURL, c.ProviderGithub.URL)
		c.Rules.Provider.ProjectKey = firstNonEmpty(c.Rules.Provider.ProjectKey, c.ProviderGithub.Project)
		c.Rules.Provider.ClientSecret = firstNonEmpty(c.Rules.Provider.ClientSecret, c.ProviderGithub.GithubPAT, envGithubPAT)
		c.ProviderGithub.URL = firstNonEmpty(c.ProviderGithub.URL, c.Rules.Provider.TokenURL)
		c.ProviderGithub.Project = firstNonEmpty(c.ProviderGithub.Project, c.Rules.Provider.ProjectKey)
		c.ProviderJira = JiraProviderConfig{}
		c.ProviderAzureDevOps = AzureDevOpsProviderConfig{}
	case "azure_devops":
		c.Rules.Provider.TokenURL = firstNonEmpty(c.Rules.Provider.TokenURL, c.ProviderAzureDevOps.URL)
		c.Rules.Provider.ProjectKey = firstNonEmpty(c.Rules.Provider.ProjectKey, c.ProviderAzureDevOps.Project)
		c.Rules.Provider.ClientSecret = firstNonEmpty(c.Rules.Provider.ClientSecret, c.ProviderAzureDevOps.PersonalAccessToken, envDevOpsPAT)
		c.ProviderAzureDevOps.URL = firstNonEmpty(c.ProviderAzureDevOps.URL, c.Rules.Provider.TokenURL)
		c.ProviderAzureDevOps.Project = firstNonEmpty(c.ProviderAzureDevOps.Project, c.Rules.Provider.ProjectKey)
		c.ProviderJira = JiraProviderConfig{}
		c.ProviderGithub = GithubProviderConfig{}
	case "jira":
		c.Rules.Provider.TokenURL = firstNonEmpty(c.Rules.Provider.TokenURL, c.ProviderJira.URL)
		c.Rules.Provider.ProjectKey = firstNonEmpty(c.Rules.Provider.ProjectKey, c.ProviderJira.Project)
		c.Rules.Provider.ClientSecret = firstNonEmpty(c.Rules.Provider.ClientSecret, c.ProviderJira.JiraAPIToken, envJiraToken)
		c.ProviderJira.URL = firstNonEmpty(c.ProviderJira.URL, c.Rules.Provider.TokenURL)
		c.ProviderJira.Project = firstNonEmpty(c.ProviderJira.Project, c.Rules.Provider.ProjectKey)
		c.ProviderGithub = GithubProviderConfig{}
		c.ProviderAzureDevOps = AzureDevOpsProviderConfig{}
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
