package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

func (d *DB) CreateSuggestion(ctx context.Context, taskID string, sugType SuggestionType, author, title, message string) (*Suggestion, error) {
	now := time.Now().UTC()
	s := &Suggestion{
		ID:        uuid.New().String(),
		TaskID:    taskID,
		Type:      sugType,
		Author:    author,
		Title:     title,
		Message:   message,
		Status:    SuggestionPending,
		CreatedAt: now,
	}

	_, err := d.conn.ExecContext(ctx,
		`INSERT INTO suggestions (id, task_id, type, author, title, message, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.TaskID, s.Type, s.Author, s.Title, s.Message, s.Status,
		s.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("creating suggestion: %w", err)
	}
	return s, nil
}

func (d *DB) ListPendingSuggestions(ctx context.Context) ([]Suggestion, error) {
	return d.listSuggestionsByStatus(ctx, SuggestionPending)
}

func (d *DB) ListSuggestionsByTask(ctx context.Context, taskID string) ([]Suggestion, error) {
	rows, err := d.conn.QueryContext(ctx,
		`SELECT id, task_id, type, author, title, message, status, created_at
		 FROM suggestions WHERE task_id=? ORDER BY created_at`, taskID)
	if err != nil {
		return nil, fmt.Errorf("listing suggestions by task: %w", err)
	}
	defer rows.Close()
	return scanSuggestions(rows)
}

func (d *DB) ListSuggestions(ctx context.Context, status SuggestionStatus) ([]Suggestion, error) {
	return d.listSuggestionsByStatus(ctx, status)
}

func (d *DB) GetSuggestion(ctx context.Context, id string) (*Suggestion, error) {
	row := d.conn.QueryRowContext(ctx,
		`SELECT id, task_id, type, author, title, message, status, created_at
		 FROM suggestions WHERE id=?`, id)

	var s Suggestion
	var createdAt string
	if err := row.Scan(&s.ID, &s.TaskID, &s.Type, &s.Author, &s.Title, &s.Message, &s.Status, &createdAt); err != nil {
		return nil, fmt.Errorf("getting suggestion: %w", err)
	}
	var parseErr error
	s.CreatedAt, parseErr = time.Parse(time.RFC3339, createdAt)
	if parseErr != nil {
		log.Printf("warning: invalid created_at for suggestion %s: %v", s.ID, parseErr)
	}
	return &s, nil
}

func (d *DB) UpdateSuggestionStatus(ctx context.Context, id string, status SuggestionStatus) error {
	_, err := d.conn.ExecContext(ctx,
		`UPDATE suggestions SET status=? WHERE id=?`, status, id)
	if err != nil {
		return fmt.Errorf("updating suggestion status: %w", err)
	}
	return nil
}

func (d *DB) listSuggestionsByStatus(ctx context.Context, status SuggestionStatus) ([]Suggestion, error) {
	rows, err := d.conn.QueryContext(ctx,
		`SELECT id, task_id, type, author, title, message, status, created_at
		 FROM suggestions WHERE status=? ORDER BY created_at`, status)
	if err != nil {
		return nil, fmt.Errorf("listing suggestions: %w", err)
	}
	defer rows.Close()
	return scanSuggestions(rows)
}

func scanSuggestions(rows interface{ Next() bool; Scan(...interface{}) error; Err() error }) ([]Suggestion, error) {
	var suggestions []Suggestion
	for rows.Next() {
		var s Suggestion
		var createdAt string
		if err := rows.Scan(&s.ID, &s.TaskID, &s.Type, &s.Author, &s.Title, &s.Message, &s.Status, &createdAt); err != nil {
			return nil, fmt.Errorf("scanning suggestion: %w", err)
		}
		var parseErr error
		s.CreatedAt, parseErr = time.Parse(time.RFC3339, createdAt)
		if parseErr != nil {
			log.Printf("warning: invalid created_at for suggestion %s: %v", s.ID, parseErr)
		}
		suggestions = append(suggestions, s)
	}
	return suggestions, rows.Err()
}
