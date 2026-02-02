package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"shop-bot/internal/domain"
	"shop-bot/internal/repository"
)

type TrackingRepo struct {
	pool *pgxpool.Pool
}

func NewTrackingRepo(pool *pgxpool.Pool) *TrackingRepo {
	return &TrackingRepo{pool: pool}
}

func (r *TrackingRepo) Create(ctx context.Context, task *domain.TrackingTask) error {
	query := `
	INSERT INTO tracking_task (user_id, shop_id, query, target_name, status)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		task.UserID,
		task.ShopID,
		task.Query,
		task.TargetName,
		task.Status,
	).Scan(&task.ID, &task.CreatedAt, &task.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create tracking task: %w", err)
	}
	return nil
}

func (r *TrackingRepo) GetActiveByUser(ctx context.Context, userID int64) ([]domain.TrackingTask, error) {
	query := `
	SELECT id, user_id, shop_id, query, target_name, status, created_at, updated_at, notified_at
	FROM tracking_task
	WHERE user_id = $1 AND satus = 'active'
	ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []domain.TrackingTask
	for rows.Next() {
		var task domain.TrackingTask
		err := rows.Scan(
			&task.ID, &task.UserID, &task.ShopID, &task.Query,
			&task.TargetName, &task.Status, &task.CreatedAt,
			&task.UpdatedAt, &task.NotifiedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return tasks, nil
}

func (r *TrackingRepo) GetAllActive(ctx context.Context) ([]domain.TrackingTask, error) {
	query := `
	SELECT id, user_id, shop_id, query, target_name, status, created_at, updated_at, notified_at
	FROM traking_tasks
	WHERE status = 'active'
	ORDER BY id ASC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all tasks: %w", err)
	}
	defer rows.Close()

	var tasks []domain.TrackingTask
	for rows.Next() {
		var task domain.TrackingTask
		err := rows.Scan(
			&task.ID, &task.UserID, &task.ShopID, &task.Query,
			&task.TargetName, &task.Status, &task.CreatedAt,
			&task.UpdatedAt, &task.NotifiedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

func (r *TrackingRepo) UpdateStatus(ctx context.Context, taskID int64, status domain.TrackingStatus) error {
	query := `
	UPDATE tracking_tasks
	SET status = $1, updated_at = NOW()
	WHERE id = $2
	`
	result, err := r.pool.Exec(ctx, query, status, taskID)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrNotFound
	}

	return nil
}

func (r *TrackingRepo) Delete(ctx context.Context, taskID int64) error {
	query := `DELETE FROM tracking_tasks WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, taskID)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrNotFound
	}

	return nil
}
