# git-enterprise-hooks

CLI helper that installs a pre-commit hook and guides task-based commit message creation.

## Prerequisites

- Go 1.22+
- Git

## Install Windows

```powershell
scoop add bucket radu103 https://github.com/radu103/radu103-bucket
scoop install git-enterprise-hooks
```

## Install MacOS with homebrew

```zsh
brew tap radu103/git-enterprise-hooks
brew trust radu103/git-enterprise-hooks
brew install git-enterprise-hooks
```

## Build

From the project root:

```powershell
.\build-release.ps1
```

This creates release artifacts in OS subfolders:

- `release/windows/git-enterprise-hooks-amd64.exe`
- `release/linux/git-enterprise-hooks-amd64`
- `release/darwin/git-enterprise-hooks-amd64`

If PowerShell execution policy blocks scripts:

```powershell
powershell -ExecutionPolicy Bypass -File .\build-release.ps1
```

### Windows / Linux / MacOS Build And Homebrew Tap Publish

1. Set the release version and build the artifact:

```powershell
$env:RELEASE_VERSION = "1.0.1"
pwsh ./build-release.ps1
```

```zsh
set RELEASE_VERSION="1.0.1"
go build -o release/darwin/git-enterprise-hooks-darwin-arm64
tar -czvf git-enterprise-hooks-arm64.tar.gz git-enterprise-hooks-arm64
```

2. Update the Homebrew tap formula to point at the new GitHub release asset:

```zsh
go build -o release/darwin/git-enterprise-hooks
cd release/darwin
tar -czvf git-enterprise-hooks.tar.gz git-enterprise-hooks
```

## Enable In A Repository

Default (Windows local repo):

1. Open PowerShell inside the target Git repository.
2. Run the Windows release binary from this project:

```powershell
C:\Work\git-enterprise-hooks\release\windows\git-enterprise-hooks-amd64.exe enable
```

Alternative (from source in this project repo):

```powershell
go run . enable
```

What `enable` does:

- Creates `git-enterprise-hooks.yaml` in repo root if missing.
- Installs a managed pre-commit hook script.
- Asks for provider type (`jira`, `azure_devops`, `github`) and stores it in config.

## One-Step Setup

Use `setup` to install the hook and configure provider values in one run:

```powershell
go run . setup
```

What `setup` does:

- Enables the managed pre-commit hook.
- Runs interactive provider setup.
- Saves provider values to `git-enterprise-hooks.yaml`.

## Runtime Environment Variables

At runtime, the CLI automatically loads `.env` and `.env.local` from the current working directory and repository root.

Supported token env vars:

- GitHub: `GITHUB_PAT`, `GITHUB_TOKEN`
- Jira: `JIRA_API_TOKEN`, `JIRA_TOKEN`
- Azure DevOps: `AZURE_DEVOPS_PAT`, `AZURE_DEVOPS_EXT_PAT`, `SYSTEM_ACCESSTOKEN`

Other supported provider env vars:

- GitHub: `GITHUB_URL`, `GITHUB_PROJECT`
- Jira: `JIRA_URL`, `JIRA_USER_EMAIL`, `JIRA_PROJECT`
- Azure DevOps: `AZURE_DEVOPS_URL`, `AZDO_URL`, `AZURE_DEVOPS_PROJECT`, `AZDO_PROJECT`

## Config Structure

- Provider blocks:
	- `provider_jira`
	- `provider_github`
	- `provider_azure_devops`
- Only the active provider block is written to config on save.

## Typical Flow


1. Run `setup` once per repository (or `enable` if you only want hook install + provider type selection).
2. Run `git commit` as usual.
3. Pre-commit asks for task search query.
4. Task selection behavior:
	 - no tasks: placeholder values are used and branch validation is skipped
	 - one task: auto-selected with console message
	 - multiple tasks in interactive terminal: selection UI is shown
	 - multiple tasks in non-interactive session: explicit error asks for interactive selection or exact key search
5. Branch validation is enforced if `rules.verify_branch_name` is set.
6. Commit message is rendered using configured template and saved into `.git/COMMIT_EDITMSG` and `.git/git-enterprise-hooks-message.txt`.

GitHub note:

- GitHub task keys are displayed as `#<number>`, but branch validation normalizes the key for pattern matching (example: `#1` is matched as `1`).

## Hook Commands

- Main pre-commit command: `hook_precommit`
- Compatibility aliases: `hook` and `hook run`

The installed hook script runs:

- `go run . hook_precommit` when local Go runtime is available
- `git-enterprise-hooks hook_precommit` as fallback

## Error Output

- CLI errors are printed once to stderr with a red `Error:` prefix.
- Hook runtime commands suppress Cobra usage spam for validation failures.

## Disable

To remove the managed hook:

```powershell
go run . disable
```
