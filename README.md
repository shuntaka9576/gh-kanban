![release](https://github.com/shuntaka9576/kanban/workflows/release/badge.svg)

# gh-kanban

A terminal kanban viewer for **GitHub Projects v2**, distributed as a [`gh` CLI](https://cli.github.com/) extension.

![gif](https://github.com/shuntaka9576/kanban/blob/master/doc/gif/kanban.gif?raw=true)

> The original `kanban` was a TUI for the now-retired GitHub Projects (Classic). This rewrite targets Projects v2, ships as a `gh` extension, and is built on top of [`cli/go-gh`](https://github.com/cli/go-gh) and [`charmbracelet/bubbletea`](https://github.com/charmbracelet/bubbletea).

## Install

```bash
gh auth login --scopes 'project'
gh extension install shuntaka9576/kanban
```

Upgrade:

```bash
gh extension upgrade kanban
```

## Usage

```bash
# by project title (server-side search, exact title match)
gh kanban view -u <USER> -p "<PROJECT TITLE>"
gh kanban view -o <ORG>  -p "<PROJECT TITLE>"

# by project number (one round-trip, fastest)
gh kanban view -o <ORG> -N 2
```

`-u` / `-o` are mutually exclusive. `-p` and `-N` are mutually exclusive — pick whichever you have at hand. Specifying `-N` skips the title search and goes straight to `projectV2(number:)`, so it lights up faster when you already know the number.

### Loading behaviour

The TUI starts immediately — the alt-screen and a spinner are visible from `t=0`. The first 100 items render as soon as GitHub responds; if the project has more, follow-up pages stream in incrementally and the column counters tick up. While paging is in flight the header shows `loaded N items, fetching more…`.

### Key bindings

| key       | action                                    |
| --------- | ----------------------------------------- |
| `h` / `l` | move focus between columns                 |
| `j` / `k` | move cursor within a column                |
| `n` / `b` | move the selected card to the next/prev Status |
| `o`       | open the selected item in the browser      |
| `O`       | open the project in the browser            |
| `y`       | yank: copy the selected Issue/PR (body + comments) as Markdown for AI context |
| `R`       | refresh from GitHub                        |
| `q`       | quit                                       |

The yank format includes the title, repository, URL, state, author, assignees, labels, body, and up to 100 comments — ready to paste into an LLM prompt. Draft issues copy only their body since they have no comments thread on GitHub.

The columns are derived from the project's **Status** SingleSelect field. Items without a Status value are grouped into a `No Status` column.

## Out of scope (for now)

- Creating / deleting items (use [`gh-p2`](https://github.com/shuntaka9576/gh-p2) for that)
- Editing fields other than Status (Iteration, Number, free text, …)
- Browsing multiple projects in one session

## Development

```bash
go build -o gh-kanban ./cmd/kanban
gh extension install .
gh kanban view -u <USER> -p "<PROJECT TITLE>"

go test ./...
```

## License

MIT
