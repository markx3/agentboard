package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (d *DB) AddComment(ctx context.Context, taskID, author, body string) (*Comment, error) {
	now := time.Now().UTC()
	c := &Comment{
		ID:        uuid.New().String(),
		TaskID:    taskID,
		Author:    author,
		Body:      body,
		CreatedAt: now,
	}

	_, err := d.conn.ExecContext(ctx,
		`INSERT INTO comments (id, task_id, author, body, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		c.ID, c.TaskID, c.Author, c.Body, c.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("adding comment: %w", err)
	}
	return c, nil
}

func (d *DB) ListComments(ctx context.Context, taskID string) ([]Comment, error) {
	rows, err := d.conn.QueryContext(ctx,
		`SELECT id, task_id, author, body, created_at
		 FROM comments WHERE task_id = ? ORDER BY created_at`, taskID)
	if err != nil {
		return nil, fmt.Errorf("listing comments: %w", err)
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var c Comment
		var createdAt string
		if err := rows.Scan(&c.ID, &c.TaskID, &c.Author, &c.Body, &createdAt); err != nil {
			return nil, fmt.Errorf("scanning comment: %w", err)
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		comments = append(comments, c)
	}
	return comments, rows.Err()
}
