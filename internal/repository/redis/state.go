package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type StateManager struct {
	client *redis.Client
	ttl    time.Duration
}

type UserState struct {
	Step      string    `json:"step"` // idle, waiting_shop, waiting_search, waiting_track
	ShopID    string    `json:"shop_id"`
	Query     string    `json:"query"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewStateManager(addr, password string, db int) (*StateManager, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &StateManager{
		client: client,
		ttl:    30 * time.Minute,
	}, nil
}

func (s *StateManager) GetState(ctx context.Context, userID int64) (*UserState, error) {
	key := fmt.Sprintf("user:%d:state", userID)

	data, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return &UserState{Step: "idle"}, nil
	}
	if err != nil {
		return nil, err
	}

	var state UserState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *StateManager) SetState(ctx context.Context, userID int64, state *UserState) error {
	key := fmt.Sprintf("user:%d:state", userID)
	state.UpdatedAt = time.Now()

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, key, data, s.ttl).Err()
}

func (s *StateManager) ClearState(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("user:%d:state", userID)
	return s.client.Del(ctx, key).Err()
}

func (s *StateManager) Close() error {
	return s.client.Close()
}
