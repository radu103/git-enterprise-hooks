package cmd

import (
	"os"
	"path/filepath"

	"github.com/radu103/git-enterprise-hooks/internal/gitutil"
	"github.com/spf13/cobra"
	"github.com/subosito/gotenv"
)

var (
	configPath string
)

var rootCmd = &cobra.Command{
	Use:   "git-enterprise-hooks",
	Short: "Enterprise commit hook assistant",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return loadDotEnvFiles()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to config yaml file")
}

func loadDotEnvFiles() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	paths := []string{
		filepath.Join(cwd, ".env"),
		filepath.Join(cwd, ".env.local"),
	}

	if repoRoot, err := gitutil.FindRepoRoot(cwd); err == nil {
		paths = append(paths,
			filepath.Join(repoRoot, ".env"),
			filepath.Join(repoRoot, ".env.local"),
		)
	}

	seen := map[string]bool{}
	for _, p := range paths {
		if seen[p] {
			continue
		}
		seen[p] = true
		if _, err := os.Stat(p); err == nil {
			if err := gotenv.Load(p); err != nil {
				return err
			}
		}
	}

	return nil
}
