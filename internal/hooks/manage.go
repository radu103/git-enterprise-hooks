package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/radu103/git-enterprise-hooks/internal/errs"
	"github.com/radu103/git-enterprise-hooks/internal/gitutil"
)

const marker = "# git-enterprise-hooks"

func Enable(repoRoot string) error {
	hooksDir, err := gitutil.HooksDir(repoRoot)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return err
	}

	hookPath := filepath.Join(hooksDir, "pre-commit")
	script := strings.Join([]string{
		"#!/bin/sh",
		marker,
		"if [ -f \"go.mod\" ] && command -v go >/dev/null 2>&1; then",
		"  go run . hook_precommit",
		"elif command -v git-enterprise-hooks >/dev/null 2>&1; then",
		"  git-enterprise-hooks hook_precommit",
		"else",
		"  echo \"git-enterprise-hooks: command not found and no local Go project runtime available\" >&2",
		"  exit 1",
		"fi",
	}, "\n") + "\n"

	if err := os.WriteFile(hookPath, []byte(script), 0o755); err != nil {
		return fmt.Errorf(errs.FmtWritePreCommitHook, err)
	}

	return nil
}

func Disable(repoRoot string) error {
	_ = removeHookScript(repoRoot)
	return nil
}

func removeHookScript(repoRoot string) error {
	hooksDir, err := gitutil.HooksDir(repoRoot)
	if err != nil {
		return err
	}
	hookPath := filepath.Join(hooksDir, "pre-commit")
	b, err := os.ReadFile(hookPath)
	if err != nil {
		return nil
	}
	if !strings.Contains(string(b), marker) {
		return nil
	}
	if err := os.Remove(hookPath); err != nil {
		return fmt.Errorf(errs.FmtRemovePreCommitHook, err)
	}
	return nil
}
