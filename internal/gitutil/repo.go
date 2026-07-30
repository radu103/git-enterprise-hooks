package gitutil

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	git "github.com/go-git/go-git/v5"
)

func FindRepoRoot(start string) (string, error) {
	cur := start
	for {
		if _, err := git.PlainOpenWithOptions(cur, &git.PlainOpenOptions{DetectDotGit: true}); err == nil {
			return cur, nil
		}
		next := filepath.Dir(cur)
		if next == cur {
			return "", errors.New("not inside a git repository")
		}
		cur = next
	}
}

func CurrentBranch(repoRoot string) (string, error) {
	cmd := exec.Command("git", "-C", repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("get current branch: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func HooksDir(repoRoot string) (string, error) {
	cmd := exec.Command("git", "-C", repoRoot, "rev-parse", "--git-path", "hooks")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("resolve hooks path: %w", err)
	}
	hooks := strings.TrimSpace(string(out))
	if filepath.IsAbs(hooks) {
		return hooks, nil
	}
	return filepath.Join(repoRoot, hooks), nil
}

func GitPath(repoRoot, rel string) (string, error) {
	cmd := exec.Command("git", "-C", repoRoot, "rev-parse", "--git-path", rel)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("resolve git path '%s': %w", rel, err)
	}
	resolved := strings.TrimSpace(string(out))
	if filepath.IsAbs(resolved) {
		return resolved, nil
	}
	return filepath.Join(repoRoot, resolved), nil
}

func EnsureIgnoreFile(path string) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return os.WriteFile(path, []byte(""), 0o644)
	}
	return nil
}
