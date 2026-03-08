package postgres

import (
	"context"
	"errors"
	"fmt"
	"shop-bot/internal/domain"
	"shop-bot/internal/repository"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// создает нового пользователя
func (r *UserRepo) Create(ctx context.Context, user *domain.User) error {
	query := `
	INSERT INTO users (telegram_id, username, first_name, last_name, selected_shop_id) 
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		user.TelegramID,
		user.Username,
		user.FirstName,
		user.LastName,
		user.SelectedShopID,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdateAt)

	if err != nil {
		if isDuplicateError(err) {
			return repository.ErrDuplicate
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// ищет пользователя по tg id
func (r *UserRepo) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error) {
	query := `
	SELECT id, telegram_id, username, first_name, last_name, selected_shop_id, created_at, updated_at
	FROM users
	WHERE telegram_id = $1
	`
	user := &domain.User{}
	err := r.pool.QueryRow(ctx, query, telegramID).Scan(
		&user.ID,
		&user.TelegramID,
		&user.Username,
		&user.FirstName,
		&user.LastName,
		&user.SelectedShopID,
		&user.CreatedAt,
		&user.UpdateAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// обновляет выбранный магазин
func (r *UserRepo) UpdateSelectedShop(ctx context.Context, telegramID int64, shopID string) error {
	query := `
	UPDATE users 
	SET selected_shop_id = $1, updated_at = NOW()
	WHERE telegram_id = $2
	`
	result, err := r.pool.Exec(ctx, query, shopID, telegramID)
	if err != nil {
		return fmt.Errorf("failed to update shop: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// проверка на уникальность
func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "unique constraint") ||
		strings.Contains(err.Error(), "duplicate key")
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	query := `
		SELECT id, telegram_id, username, first_name, last_name,
		selected_shop_id, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &domain.User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.TelegramID, &user.Username, &user.FirstName,
		&user.LastName, &user.SelectedShopID,
		&user.CreatedAt, &user.UpdateAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return user, nil
}
