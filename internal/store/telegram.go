package store

import (
	"context"
)

func (s *Store) ClaimTelegramUpdate(ctx context.Context, updateID int64) (bool, error) {
	result, err := s.DB.ExecContext(ctx, `
		INSERT INTO processed_telegram_updates(update_id, processed_at)
		VALUES (?, ?)
		ON CONFLICT(update_id) DO NOTHING`, updateID, nowUTC())
	if err != nil {
		return false, err
	}
	affected, _ := result.RowsAffected()
	return affected == 1, nil
}
