# gotodo

A keyboard-first terminal task manager built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

No mice. No config files. No ceremony.

---

## Features

- **Vim-style navigation** — `j`/`k` to move, `J`/`K` to reorder
- **Subtasks** — nest tasks under any other task with `s`
- **Headings** — create named groups to organize tasks with `g`
- **Projects** — tag tasks with `+project` when adding (e.g. `fix auth +backend`)
- **Promote / Demote** — `H`/`L` to change nesting depth
- **Progress** — live task counter with visual progress bar
- **Persistent** — saves to JSON, auto-creates storage directory
- **Command bar** — `:` for write, quit, toggle, and file switching
- **Zero config** — just run it

---

## Keybindings

### Navigation

| Key | Action |
|-----|--------|
| `j` / `k` or `↑` / `↓` | Move cursor |
| `J` / `K` | Reorder task up / down |
| `H` / `L` | Promote / demote (change indent level) |

### Tasks

| Key | Action |
|-----|--------|
| `a` | Add task (use `+tag` for project grouping) |
| `s` | Add subtask under cursor |
| `g` | Add heading / group |
| `space` | Toggle done |
| `d` / `backspace` | Delete task |
| `t` | Toggle show done / open |

### Meta

| Key | Action |
|-----|--------|
| `:` | Command bar |
| `?` | Help modal |
| `q` / `Ctrl+C` | Quit |

---

## Command Bar

Press `:` to open, then type:

| Command | Action |
|---------|--------|
| `:w` | Save |
| `:q` | Quit |
| `:toggle done` | Toggle done / all view |
| `:set file=~/path.json` | Switch save file (reloads) |

---

## Storage

Default: `~/.config/gotodo/tasks.json`

Override with a flag or environment variable:

```bash
gotodo --file ./tasks.json
```

```bash
GOTODO_FILE=~/tasks.json gotodo
```

Priority: `--file` flag > `GOTODO_FILE` env > default path.

---

## Install

```bash
git clone https://github.com/drakeafk/gotodo.git
cd gotodo
go build -o gotodo .
./gotodo
```

---

## Project Layout

```
.
├── main.go    # startup, flags, path resolution
├── model.go   # Bubble Tea state machine + rendering
├── store.go   # JSON persistence (atomic write)
├── task.go    # Task struct
├── go.mod
└── go.sum
```

---

## Development

```bash
go fmt ./...
go vet ./...
go run .
```

---

## License

MIT. Use it, fork it, break it.
