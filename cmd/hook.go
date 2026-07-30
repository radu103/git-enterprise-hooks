package cmd

import (
	"context"
	"os"

	"github.com/radu103/git-enterprise-hooks/internal/config"
	"github.com/radu103/git-enterprise-hooks/internal/gitutil"
	"github.com/radu103/git-enterprise-hooks/internal/workflow"
	"github.com/spf13/cobra"
)

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Hook runtime commands",
	RunE:  runHookFlow,
}

var hookPreCommitCmd = &cobra.Command{
	Use:   "hook_precommit",
	Short: "Run pre-commit hook flow",
	RunE:  runHookFlow,
}

var hookRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run pre-commit flow",
	RunE:  runHookFlow,
}

func runHookFlow(cmd *cobra.Command, args []string) error {
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
	return workflow.RunPreCommit(context.Background(), repoRoot, cfg, cfgPath)
}

func init() {
	hookCmd.AddCommand(hookRunCmd)
	rootCmd.AddCommand(hookCmd)
	rootCmd.AddCommand(hookPreCommitCmd)
}
