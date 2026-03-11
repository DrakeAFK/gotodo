package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

func resolvePath(flagPath string) (string, error) {
	if flagPath != "" {
		return expandHome(flagPath)
	}
	if env := os.Getenv("GOTODO_FILE"); env != "" {
		return expandHome(env)
	}
	return DefaultStorePath()
}

func expandHome(p string) (string, error) {
	if len(p) > 0 && p[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if p == "~" {
			return home, nil
		}
		if len(p) >= 2 && (p[1] == '/' || p[1] == '\\') {
			return filepath.Join(home, p[2:]), nil
		}
	}
	return p, nil
}

func main() {
	var fileFlag string
	flag.StringVar(&fileFlag, "file", "", "path to tasks.json (overrides GOTODO_FILE and default)")
	flag.Parse()

	path, err := resolvePath(fileFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to determine storage path:", err)
		os.Exit(1)
	}

	store := Store{Path: path}
	st, err := store.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load tasks:", err)
		os.Exit(1)
	}

	m := NewModel(store, st)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "boom:", err)
		os.Exit(1)
	}
}