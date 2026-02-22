# Personal Preferences

## Architecture: Agents, Skills, and Scripts

- **Agents** handle singularly focused tasks. Each agent owns one domain of expertise.
- **Skills** compose agents and scripts into higher-level workflows. A skill may invoke multiple agents or scripts to accomplish a broader goal.
- **Scripts** live in `~/.claude/scripts/<name>/` as self-contained Go modules. Each script:
  - Has its own directory with a `go.mod` and `main.go`
  - Does the work of exactly one agent or skill — no more
  - Is referenceable by a skill or agent definition
  - Should be informed by the `golang-expert` agent when being written or reviewed

## Scripting Language

- Prefer **Go** for all scripts in `~/.claude/scripts/`.
- Use the `golang-expert` agent to guide idiomatic Go patterns, error handling, and structure.
- Existing shell scripts (`.sh`) may remain but new scripts should be Go.

## Go Version Policy

- **Scripts and commands** (`~/.claude/scripts/`, CLIs, standalone tools): use the latest stable Go version in `go.mod`.
- **Modules and libraries** (shared packages, importable code): use the minimum Go version required by the features actually used. This maximizes compatibility for consumers.

## Credentials

- Never hardcode or embed credentials in scripts or config files.
- Prefer standard user credential locations. When credentials are needed, prompt me to choose which to use from:
  - `~/.ssh/` — SSH keys
  - `~/.aws/` — AWS credentials and config
  - `~/.kube/` — Kubernetes kubeconfigs
  - `~/.netrc` — Machine tokens (GitLab, Confluence, Jira, Harbor, etc.)
  - `~/.docker/config.json` — Container registry auth
  - `~/.config/` — XDG-standard application configs
- Scripts should accept credential paths as flags or environment variables, not assume a single source.
