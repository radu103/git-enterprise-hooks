package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/radu103/git-enterprise-hooks/internal/config"
	"github.com/radu103/git-enterprise-hooks/internal/errs"
	"github.com/radu103/git-enterprise-hooks/internal/gitutil"
	"github.com/radu103/git-enterprise-hooks/internal/hooks"
	"github.com/radu103/git-enterprise-hooks/internal/ui"
	"github.com/spf13/cobra"
)

var enableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable enterprise pre-commit hook",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		repoRoot, err := gitutil.FindRepoRoot(cwd)
		if err != nil {
			return err
		}
		cfg, cfgPath, err := config.LoadOrCreate(repoRoot, configPath)
		if err != nil {
			return err
		}

		providerTypeInput, err := ui.Ask("Provider type (jira/azure_devops/github)", cfg.Rules.Provider.Type)
		if err != nil {
			return err
		}

		providerType := normalizeProviderType(providerTypeInput)
		if providerType == "" {
			return fmt.Errorf(errs.FmtUnsupportedProviderTypeAllowed, providerTypeInput)
		}

		cfg.Rules.Provider.Type = providerType
		if err := config.Save(cfgPath, cfg); err != nil {
			return err
		}

		if err := hooks.Enable(repoRoot); err != nil {
			return err
		}

		fmt.Printf("Enabled hooks (pre-commit, prepare-commit-msg) in %s\n", repoRoot)
		fmt.Printf("Config: %s\n", cfgPath)
		fmt.Printf("Provider type: %s\n", providerType)
		printProviderSetupInstructions(providerType)

		setupNow, err := ui.Confirm("Do you want to set up provider values in config now")
		if err != nil {
			return err
		}
		if setupNow {
			if err := promptAndSaveProviderValues(cfgPath, &cfg); err != nil {
				return err
			}
			fmt.Printf("Saved provider settings to %s.\n", cfgPath)
			fmt.Println("Then run: git commit")
		} else {
			fmt.Println("Next: set up provider config first (in config file or env vars), then run 'git commit'.")
		}
		return nil
	},
}

func promptAndSaveProviderValues(cfgPath string, cfg *config.Config) error {
	ptype := cfg.Rules.Provider.Type
	switch ptype {
	case "jira":
		url, err := ui.Ask("Jira URL", cfg.ProviderJira.URL)
		if err != nil {
			return err
		}
		project, err := ui.Ask("Jira Project", cfg.ProviderJira.Project)
		if err != nil {
			return err
		}
		userEmail, err := ui.Ask("Jira user email", cfg.ProviderJira.UserEmail)
		if err != nil {
			return err
		}
		token, err := ui.AskPassword("Jira API token (leave empty to use env or existing)")
		if err != nil {
			return err
		}
		cfg.ProviderJira.URL = url
		cfg.ProviderJira.Project = project
		cfg.ProviderJira.UserEmail = userEmail
		if strings.TrimSpace(token) != "" {
			cfg.ProviderJira.JiraAPIToken = token
		}
	case "github":
		url, err := ui.Ask("GitHub URL", cfg.ProviderGithub.URL)
		if err != nil {
			return err
		}
		project, err := ui.Ask("GitHub project", cfg.ProviderGithub.Project)
		if err != nil {
			return err
		}
		token, err := ui.AskPassword("GitHub PAT (leave empty to use env or existing)")
		if err != nil {
			return err
		}
		cfg.ProviderGithub.URL = url
		cfg.ProviderGithub.Project = project
		if strings.TrimSpace(token) != "" {
			cfg.ProviderGithub.GithubPAT = token
		}
	case "azure_devops":
		url, err := ui.Ask("Azure DevOps URL", cfg.ProviderAzureDevOps.URL)
		if err != nil {
			return err
		}
		project, err := ui.Ask("Azure DevOps project", cfg.ProviderAzureDevOps.Project)
		if err != nil {
			return err
		}
		token, err := ui.AskPassword("Azure DevOps PAT (leave empty to use env or existing)")
		if err != nil {
			return err
		}
		cfg.ProviderAzureDevOps.URL = url
		cfg.ProviderAzureDevOps.Project = project
		if strings.TrimSpace(token) != "" {
			cfg.ProviderAzureDevOps.PersonalAccessToken = token
		}
	default:
		// unknown provider; nothing to prompt
	}

	if err := config.Save(cfgPath, *cfg); err != nil {
		return err
	}
	return nil
}

func init() {
	rootCmd.AddCommand(enableCmd)
}

func normalizeProviderType(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	switch v {
	case "jira":
		return "jira"
	case "azure_devops", "azure-devops", "devops":
		return "azure_devops"
	case "github":
		return "github"
	default:
		return ""
	}
}

func printProviderSetupInstructions(providerType string) {
	fmt.Println("Setup guidance:")
	switch providerType {
	case "github":
		fmt.Println("- Required: provider_github.url and provider_github.project")
		fmt.Println("- Token: provider_github.github_pat or env GITHUB_PAT/GITHUB_TOKEN")
	case "azure_devops":
		fmt.Println("- Required: provider_azure_devops.url and provider_azure_devops.project")
		fmt.Println("- Token: provider_azure_devops.personal_access_token or env AZURE_DEVOPS_PAT/AZURE_DEVOPS_EXT_PAT/SYSTEM_ACCESSTOKEN")
	default:
		fmt.Println("- Required: provider_jira.url, provider_jira.project, provider_jira.user_email")
		fmt.Println("- Token: provider_jira.jira_api_token or env JIRA_API_TOKEN/JIRA_TOKEN")
	}
	fmt.Println("- Missing values will be prompted during pre-commit run.")
}
