package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type diskState struct {
	NextID int    `json:"next_id"`
	Tasks  []Task `json:"tasks"`
}

type Store struct {
	Path string
}

func DefaultStorePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "gotodo", "tasks.json"), nil
}

func (s Store) Load() (diskState, error) {
	var st diskState

	b, err := os.ReadFile(s.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return diskState{NextID: 1, Tasks: []Task{}}, nil
		}
		return st, err
	}

	if err := json.Unmarshal(b, &st); err != nil {
		return st, err
	}

	// Migrate legacy "Project: Task" names to actual Projects
	for i, t := range st.Tasks {
		if t.ParentID == 0 && t.Project == "" {
			importances := strings.SplitN(t.Title, ": ", 2)
			if len(importances) == 2 {
				st.Tasks[i].Project = strings.TrimSpace(importances[0])
				st.Tasks[i].Title = strings.TrimSpace(importances[1])
			}
		}
	}

	if st.NextID <= 0 {
		st.NextID = 1
	}
	return st, nil
}

func (s Store) Save(st diskState) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return err
	}

	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')

	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}