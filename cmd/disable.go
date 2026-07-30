package cmd

import (
	"errors"
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

var disableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable enterprise hooks (pre-commit, prepare-commit-msg)",
	RunE: func(cmd *cobra.Command, args []string) error {
		answer, err := ui.Ask("Disable hook (yes/no)", "yes")
		if err != nil {
			return err
		}
		answer = strings.ToLower(strings.TrimSpace(answer))
		ok := answer == "yes" || answer == "y"
		if !ok {
			fmt.Println("Aborted.")
			return nil
		}
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		repoRoot, err := gitutil.FindRepoRoot(cwd)
		if err != nil {
			return err
		}
		if err := hooks.Disable(repoRoot); err != nil {
			return err
		}

		deleteConfig, err := ui.Confirm("Delete config file too (default: no)")
		if err != nil {
			return err
		}
		if deleteConfig {
			cfgPath := config.ResolvePath(repoRoot, configPath)
			if err := os.Remove(cfgPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf(errs.FmtDeleteConfigFile, err)
			}
			fmt.Printf("Deleted config: %s\n", cfgPath)
		} else {
			fmt.Println("Config file kept.")
		}

		fmt.Println("Cleanup complete.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(disableCmd)
}
