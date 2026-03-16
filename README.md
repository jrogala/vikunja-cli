# vikunja-cli

A fast CLI client for [Vikunja](https://vikunja.io) task management. Compatible with Vikunja v2.

## Install

```bash
go install github.com/jrogala/vikunja-cli@latest
```

## Configure

Interactive setup (recommended):

```bash
vikunja-cli setup
```

This validates the connection and writes to `~/.config/vikunja-cli/config.yaml`.

Alternatively, use environment variables:

```bash
export VIKUNJA_URL="https://your-instance.com/api/v1"
export VIKUNJA_TOKEN="tk_your_api_token"
```

## Usage

```
vikunja-cli setup                      # Interactive configuration
vikunja-cli task list                  # List incomplete tasks
vikunja-cli task list --all            # Include completed tasks
vikunja-cli task list --done           # Only completed tasks
vikunja-cli task list -p 2             # Filter by project ID
vikunja-cli task list -s priority      # Sort by priority
vikunja-cli task list --search "bug"   # Search by title
vikunja-cli task get <id>              # Get task details
vikunja-cli task add "Fix login bug"   # Create task in Inbox
vikunja-cli task add -p 2 "Deploy"     # Create in project 2
vikunja-cli task add --priority 3 "X"  # Create with high priority
vikunja-cli task add --due-date 2026-03-20  # Create with due date
vikunja-cli task done <id>             # Mark task complete
vikunja-cli task delete <id>           # Delete task

vikunja-cli project list               # List all projects

vikunja-cli --json task list           # JSON output (for LLM/scripts)
```

### Aliases

- `task` / `tasks` / `t`
- `project` / `projects` / `p`
- `list` / `ls` / `l`

### Priority levels

| Value | Label  |
|-------|--------|
| 0     | None   |
| 1     | Low    |
| 2     | Medium |
| 3     | High   |
| 4     | Urgent |

## License

MIT
