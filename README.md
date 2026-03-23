# vikunja-cli

CLI for Vikunja task and project management.

## Setup

Set env vars or run setup:

```bash
export VIKUNJA_URL=https://vikunja.example.com/api/v1
export VIKUNJA_TOKEN=tk_your_api_token
```

Or configure interactively:

```bash
vikunja setup
```

Config stored at `~/.config/vikunja-cli/config.yaml`.

## Commands

| Command | Description |
|---|---|
| `setup` | Configure URL and token interactively |
| `project list` | List all projects |
| `task list` | List tasks (flags: --all, --done, -p PROJECT, -s SORT, --search) |
| `task get` | Show details of a specific task |
| `task add` | Create a new task (flags: -p PROJECT, --priority, --due-date) |
| `task done` | Mark a task as completed |
| `task edit` | Update a task's fields |
| `task delete` | Delete a task |

Aliases: `task`/`tasks`/`t`, `project`/`projects`/`p`, `list`/`ls`/`l`.

## Examples

```bash
# List all projects
vikunja project list

# Add a task to a project with priority
vikunja task add -p 2 --priority 3 "Fix kitchen light"

# List open tasks in a project
vikunja task list -p 2

# Mark a task as done
vikunja task done 42

# Search tasks
vikunja task list --search "deploy"
```

## JSON Output

All commands support `--json` for machine-readable output.
