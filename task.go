package main

import "time"

type Task struct {
	ID        int       `json:"id"`
	ParentID  int       `json:"parent_id,omitempty"` // 0 means root task
	Project   string    `json:"project,omitempty"`   // parsed from +project
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	IsHeading bool      `json:"is_heading,omitempty"` // renders as a group separator
	CreatedAt time.Time `json:"created_at"`
	DoneAt    time.Time `json:"done_at,omitempty"`
}