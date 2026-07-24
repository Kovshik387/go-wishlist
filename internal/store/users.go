package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/example/wishtrack/internal/platform"
)

func (s *Store) UpsertTelegramUser(ctx context.Context, tg TelegramUser) (User, error) {
	now := nowUTC()
	displayName := strings.TrimSpace(strings.TrimSpace(tg.FirstName) + " " + strings.TrimSpace(tg.LastName))
	if displayName == "" {
		displayName = tg.Username
	}
	if displayName == "" {
		displayName = "Друг"
	}
	language := tg.LanguageCode
	if language == "" {
		language = "ru"
	}
	id := platform.UUID()
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO users (
			id, telegram_id, username, display_name, avatar_url, language_code,
			timezone, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, 'Europe/Moscow', ?, ?)
		ON CONFLICT(telegram_id) DO UPDATE SET
			username = excluded.username,
			avatar_url = CASE WHEN excluded.avatar_url <> '' THEN excluded.avatar_url ELSE users.avatar_url END,
			language_code = excluded.language_code,
			updated_at = excluded.updated_at`,
		id, tg.ID, tg.Username, displayName, tg.PhotoURL, language, now, now)
	if err != nil {
		return User{}, fmt.Errorf("upsert telegram user: %w", err)
	}
	return s.UserByTelegramID(ctx, tg.ID)
}

func (s *Store) UserByTelegramID(ctx context.Context, telegramID int64) (User, error) {
	return scanUser(s.DB.QueryRowContext(ctx, userSelect+` WHERE deleted_at IS NULL AND telegram_id = ?`, telegramID))
}

func (s *Store) UserByID(ctx context.Context, id string) (User, error) {
	return scanUser(s.DB.QueryRowContext(ctx, userSelect+` WHERE deleted_at IS NULL AND id = ?`, id))
}

func (s *Store) PublicUserByID(ctx context.Context, id, viewerID string) (User, bool, error) {
	user, err := s.UserByID(ctx, id)
	if err != nil {
		return User{}, false, err
	}
	user.TelegramID = 0
	var following int
	err = s.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM follows WHERE follower_id = ? AND followee_id = ?`,
		viewerID, id).Scan(&following)
	return user, following > 0, err
}

func (s *Store) PatchUser(ctx context.Context, id, displayName, timezone string, onboarding *bool) (User, error) {
	if strings.TrimSpace(displayName) == "" {
		return User{}, fmt.Errorf("display name is required")
	}
	if timezone == "" {
		timezone = "Europe/Moscow"
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return User{}, fmt.Errorf("invalid timezone")
	}
	complete := 0
	if onboarding != nil && *onboarding {
		complete = 1
	}
	_, err := s.DB.ExecContext(ctx, `
		UPDATE users SET display_name = ?, timezone = ?,
			onboarding_completed = CASE WHEN ? = 1 THEN 1 ELSE onboarding_completed END,
			updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`,
		strings.TrimSpace(displayName), timezone, complete, nowUTC(), id)
	if err != nil {
		return User{}, err
	}
	return s.UserByID(ctx, id)
}

func (s *Store) SetBotWriteAllowed(ctx context.Context, telegramID int64, allowed bool) error {
	_, err := s.DB.ExecContext(ctx,
		`UPDATE users SET bot_write_allowed = ?, updated_at = ? WHERE telegram_id = ? AND deleted_at IS NULL`,
		boolInt(allowed), nowUTC(), telegramID)
	return err
}

func (s *Store) DeleteAccount(ctx context.Context, id string) error {
	now := nowUTC()
	return withTx(ctx, s.DB, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE users SET
				telegram_id = -ABS(telegram_id),
				username = '',
				display_name = 'Удалённый пользователь',
				avatar_url = '',
				bot_write_allowed = 0,
				updated_at = ?,
				deleted_at = ?
			WHERE id = ? AND deleted_at IS NULL`, now, now, id); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE user_sessions SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`, now, id)
		return err
	})
}

const userSelect = `SELECT id, telegram_id, username, display_name, avatar_url, language_code,
	timezone, bot_write_allowed, onboarding_completed, created_at FROM users`

type rowScanner interface {
	Scan(...any) error
}

func scanUser(row rowScanner) (User, error) {
	var user User
	var botWrite, onboarding int
	err := row.Scan(&user.ID, &user.TelegramID, &user.Username, &user.DisplayName, &user.AvatarURL,
		&user.LanguageCode, &user.Timezone, &botWrite, &onboarding, &user.CreatedAt)
	if err != nil {
		return User{}, isNoRows(err)
	}
	user.BotWriteAllowed = botWrite != 0
	user.OnboardingCompleted = onboarding != 0
	return user, nil
}

func (s *Store) CreateSession(ctx context.Context, userID, refreshToken string, expiresAt time.Time) error {
	sum := sha256.Sum256([]byte(refreshToken))
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO user_sessions(id, user_id, refresh_token_hash, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		platform.UUID(), userID, hex.EncodeToString(sum[:]), expiresAt.UTC().Format(time.RFC3339Nano), nowUTC())
	return err
}

func (s *Store) RotateSession(ctx context.Context, oldToken, newToken string, newExpiry time.Time) (User, error) {
	oldHash := sha256.Sum256([]byte(oldToken))
	newHash := sha256.Sum256([]byte(newToken))
	var user User
	err := withTx(ctx, s.DB, func(tx *sql.Tx) error {
		row := tx.QueryRowContext(ctx, userSelect+`
			JOIN user_sessions s ON s.user_id = users.id
			WHERE s.refresh_token_hash = ? AND s.revoked_at IS NULL
			  AND s.expires_at > ? AND users.deleted_at IS NULL`,
			hex.EncodeToString(oldHash[:]), nowUTC())
		var botWrite, onboarding int
		if err := row.Scan(&user.ID, &user.TelegramID, &user.Username, &user.DisplayName, &user.AvatarURL,
			&user.LanguageCode, &user.Timezone, &botWrite, &onboarding, &user.CreatedAt); err != nil {
			return isNoRows(err)
		}
		user.BotWriteAllowed = botWrite != 0
		user.OnboardingCompleted = onboarding != 0
		now := nowUTC()
		result, err := tx.ExecContext(ctx,
			`UPDATE user_sessions SET revoked_at = ? WHERE refresh_token_hash = ? AND revoked_at IS NULL`,
			now, hex.EncodeToString(oldHash[:]))
		if err != nil {
			return err
		}
		affected, _ := result.RowsAffected()
		if affected != 1 {
			return ErrConflict
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO user_sessions(id, user_id, refresh_token_hash, expires_at, created_at)
			VALUES (?, ?, ?, ?, ?)`,
			platform.UUID(), user.ID, hex.EncodeToString(newHash[:]),
			newExpiry.UTC().Format(time.RFC3339Nano), now)
		return err
	})
	return user, err
}

func (s *Store) RevokeSession(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	sum := sha256.Sum256([]byte(token))
	_, err := s.DB.ExecContext(ctx,
		`UPDATE user_sessions SET revoked_at = ? WHERE refresh_token_hash = ? AND revoked_at IS NULL`,
		nowUTC(), hex.EncodeToString(sum[:]))
	return err
}
