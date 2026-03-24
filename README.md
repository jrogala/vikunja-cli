# vikunja-cli

CLI for Vikunja task and project management.

## Install

```bash
go install github.com/jrogala/vikunja-cli@latest
```

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
$ vikunja-cli project list
ID  TITLE          FAV  ARCHIVED
1   Inbox
2   Work Projects  *
3   Personal       *

$ vikunja-cli task list -p 2
ID   DONE  PRIORITY  DUE        TITLE
1    [ ]   HIGH      today      Fix kitchen light
2    [x]   -         Mar 02     Complete documentation
3    [ ]   URGENT    tomorrow   Deploy to production

$ vikunja-cli task add -p 2 --priority 3 "Review PR #42"
Created task #4: Review PR #42

$ vikunja-cli task done 42
Completed task #42: Review PR #42
```

## JSON Output

All commands support `--json` for machine-readable output.
