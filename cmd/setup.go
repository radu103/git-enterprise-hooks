package cmd

import (
	"fmt"
	"os"

	"github.com/radu103/git-enterprise-hooks/internal/config"
	"github.com/radu103/git-enterprise-hooks/internal/gitutil"
	"github.com/radu103/git-enterprise-hooks/internal/hooks"
	"github.com/radu103/git-enterprise-hooks/internal/workflow"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Enable hook and set up provider configuration",
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

		if err := hooks.Enable(repoRoot); err != nil {
			return err
		}

		if err := workflow.SetupProviderConfig(&cfg); err != nil {
			return err
		}

		if err := config.Save(cfgPath, cfg); err != nil {
			return err
		}

		fmt.Printf("Hook enabled in %s\n", repoRoot)
		fmt.Printf("Provider setup complete. Config updated: %s\n", cfgPath)
		fmt.Println("Next: run 'git commit' to continue with task-assisted commit message flow.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
