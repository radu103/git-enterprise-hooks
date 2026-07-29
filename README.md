# git-enterprise-hooks

CLI helper that installs a pre-commit hook and guides task-based commit message creation.

## Prerequisites

- Go 1.22+
- Git

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

## Runtime Environment Variables

At runtime, the CLI automatically loads `.env` and `.env.local` from the current working directory and repository root.

Supported token env vars:

- GitHub: `GITHUB_PAT`, `GITHUB_TOKEN`
- Jira: `JIRA_API_TOKEN`, `JIRA_TOKEN`
- Azure DevOps: `AZURE_DEVOPS_PAT`, `AZURE_DEVOPS_EXT_PAT`, `SYSTEM_ACCESSTOKEN`

## Typical Flow

1. Run `enable` once per repository.
2. Run `git commit` as usual.
3. On first run, provider config is prompted and saved to `git-enterprise-hooks.yaml`.
4. Task selection and commit message formatting run as part of pre-commit.

## Disable

To remove the managed hook:

```powershell
go run . disable
```
