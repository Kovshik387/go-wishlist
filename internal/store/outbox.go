package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/example/wishtrack/internal/platform"
)

type outboxPayload struct {
	WishID     string `json:"wishId"`
	WishlistID string `json:"wishlistId"`
	AuthorID   string `json:"authorId"`
}

type deliveryPayload struct {
	AuthorName string   `json:"authorName"`
	WishTitles []string `json:"wishTitles"`
	WishCount  int      `json:"wishCount"`
	PriceMinor *int64   `json:"priceMinor,omitempty"`
	Currency   string   `json:"currency,omitempty"`
	DeepLink   string   `json:"deepLink"`
}

func (s *Store) PrepareOutbox(ctx context.Context, digestWindow time.Duration, botUsername, appShortName string, limit int) (int, error) {
	if limit < 1 {
		limit = 20
	}
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, payload, created_at
		FROM outbox_events
		WHERE status = 'pending' AND available_at <= ?
		ORDER BY created_at
		LIMIT ?`, nowUTC(), limit)
	if err != nil {
		return 0, err
	}
	type event struct {
		ID        string
		Payload   outboxPayload
		CreatedAt time.Time
	}
	var events []event
	for rows.Next() {
		var item event
		var raw, created string
		if err := rows.Scan(&item.ID, &raw, &created); err != nil {
			_ = rows.Close()
			return 0, err
		}
		if err := json.Unmarshal([]byte(raw), &item.Payload); err != nil {
			_ = rows.Close()
			return 0, err
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		events = append(events, item)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}

	processed := 0
	seenAuthors := map[string]bool{}
	for _, item := range events {
		if seenAuthors[item.Payload.AuthorID] {
			continue
		}
		seenAuthors[item.Payload.AuthorID] = true
		count, err := s.prepareAuthorDigest(ctx, item, digestWindow, botUsername, appShortName)
		if err != nil {
			return processed, err
		}
		processed += count
	}
	return processed, nil
}

func (s *Store) prepareAuthorDigest(ctx context.Context, first struct {
	ID        string
	Payload   outboxPayload
	CreatedAt time.Time
}, digestWindow time.Duration, botUsername, appShortName string) (int, error) {
	type groupedEvent struct {
		ID      string
		Payload outboxPayload
	}
	var group []groupedEvent
	err := withTx(ctx, s.DB, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
			UPDATE outbox_events SET status = 'processing', attempts = attempts + 1
			WHERE id = ? AND status = 'pending'`, first.ID)
		if err != nil {
			return err
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return ErrConflict
		}

		rows, err := tx.QueryContext(ctx, `
			SELECT id, payload
			FROM outbox_events
			WHERE event_type = 'wish.created'
			  AND status IN ('pending', 'processing')
			  AND json_extract(payload, '$.authorId') = ?
			  AND created_at <= ?
			ORDER BY created_at`,
			first.Payload.AuthorID, first.CreatedAt.Add(digestWindow).Format(time.RFC3339Nano))
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item groupedEvent
			var raw string
			if err := rows.Scan(&item.ID, &raw); err != nil {
				return err
			}
			if err := json.Unmarshal([]byte(raw), &item.Payload); err != nil {
				return err
			}
			group = append(group, item)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		for _, item := range group {
			if _, err := tx.ExecContext(ctx, `
				UPDATE outbox_events SET status = 'processing', attempts = attempts + 1
				WHERE id = ? AND status = 'pending'`, item.ID); err != nil {
				return err
			}
		}
		return nil
	})
	if errors.Is(err, ErrConflict) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if len(group) == 0 {
		return 0, nil
	}

	eventIDs := make([]string, 0, len(group))
	wishIDs := make([]string, 0, len(group))
	for _, item := range group {
		eventIDs = append(eventIDs, item.ID)
		wishIDs = append(wishIDs, item.Payload.WishID)
	}
	return len(group), s.createDigestDeliveries(ctx, first.Payload.AuthorID, eventIDs, wishIDs, botUsername, appShortName)
}

func (s *Store) createDigestDeliveries(ctx context.Context, authorID string, eventIDs, wishIDs []string, botUsername, appShortName string) error {
	return withTx(ctx, s.DB, func(tx *sql.Tx) error {
		var authorName string
		if err := tx.QueryRowContext(ctx,
			`SELECT display_name FROM users WHERE id = ?`, authorID).Scan(&authorName); err != nil {
			return err
		}
		titles := make([]string, 0, len(wishIDs))
		var price *int64
		var currency, publicToken string
		for index, wishID := range wishIDs {
			var title string
			var nullablePrice sql.NullInt64
			if err := tx.QueryRowContext(ctx, `
				SELECT wi.title, wi.price_minor, wi.currency, w.public_token
				FROM wishes wi JOIN wishlists w ON w.id = wi.wishlist_id
				WHERE wi.id = ?`, wishID).Scan(&title, &nullablePrice, &currency, &publicToken); err != nil {
				return err
			}
			titles = append(titles, title)
			if index == 0 && nullablePrice.Valid {
				value := nullablePrice.Int64
				price = &value
			}
		}
		deepLink := fmt.Sprintf("https://t.me/%s/%s?startapp=wishlist_%s",
			botUsername, appShortName, publicToken)
		payload, _ := json.Marshal(deliveryPayload{
			AuthorName: authorName, WishTitles: titles, WishCount: len(titles),
			PriceMinor: price, Currency: currency, DeepLink: deepLink,
		})
		rows, err := tx.QueryContext(ctx, `
			SELECT u.id
			FROM follows f
			JOIN users u ON u.id = f.follower_id
			LEFT JOIN follow_notification_settings fs
				ON fs.follower_id = f.follower_id AND fs.followee_id = f.followee_id
			LEFT JOIN notification_preferences np ON np.user_id = u.id
			WHERE f.followee_id = ?
			  AND u.deleted_at IS NULL
			  AND u.bot_write_allowed = 1
			  AND COALESCE(fs.muted, 0) = 0
			  AND COALESCE(np.enabled, 1) = 1
			  AND COALESCE(np.new_wishes, 1) = 1`, authorID)
		if err != nil {
			return err
		}
		var followers []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				_ = rows.Close()
				return err
			}
			followers = append(followers, id)
		}
		if err := rows.Close(); err != nil {
			return err
		}
		eventKey := strings.Join(eventIDs, ",")
		for _, followerID := range followers {
			sum := sha256.Sum256([]byte(followerID + ":" + eventKey))
			_, err := tx.ExecContext(ctx, `
				INSERT INTO notification_deliveries(
					id, event_id, user_id, dedup_key, payload, next_attempt_at, created_at
				) VALUES (?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(dedup_key) DO NOTHING`,
				platform.UUID(), eventIDs[0], followerID, hex.EncodeToString(sum[:]), string(payload),
				nowUTC(), nowUTC())
			if err != nil {
				return err
			}
		}
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(eventIDs)), ",")
		args := make([]any, 0, len(eventIDs)+2)
		args = append(args, nowUTC())
		for _, id := range eventIDs {
			args = append(args, id)
		}
		_, err = tx.ExecContext(ctx, `
			UPDATE outbox_events SET status = 'processed', processed_at = ?
			WHERE id IN (`+placeholders+`)`, args...)
		return err
	})
}

func (s *Store) ClaimDelivery(ctx context.Context) (Delivery, error) {
	var delivery Delivery
	err := withTx(ctx, s.DB, func(tx *sql.Tx) error {
		var raw string
		err := tx.QueryRowContext(ctx, `
			SELECT d.id, d.event_id, d.user_id, u.telegram_id, d.payload, d.attempts, u.timezone
			FROM notification_deliveries d
			JOIN users u ON u.id = d.user_id
			WHERE d.status IN ('pending', 'failed')
			  AND d.next_attempt_at <= ?
			  AND u.bot_write_allowed = 1
			ORDER BY d.created_at
			LIMIT 1`, nowUTC()).Scan(
			&delivery.ID, &delivery.EventID, &delivery.UserID, &delivery.TelegramID,
			&raw, &delivery.Attempts, &delivery.Timezone,
		)
		if err != nil {
			return isNoRows(err)
		}
		result, err := tx.ExecContext(ctx, `
			UPDATE notification_deliveries
			SET status = 'processing', attempts = attempts + 1
			WHERE id = ? AND status IN ('pending', 'failed')`, delivery.ID)
		if err != nil {
			return err
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return ErrConflict
		}
		delivery.Payload = json.RawMessage(raw)
		return nil
	})
	if err != nil {
		return Delivery{}, err
	}
	prefs, err := s.NotificationPreferences(ctx, delivery.UserID)
	if err != nil {
		return Delivery{}, err
	}
	delivery.Prefs = prefs
	return delivery, nil
}

func (s *Store) MarkDeliverySent(ctx context.Context, id string) error {
	_, err := s.DB.ExecContext(ctx, `
		UPDATE notification_deliveries SET status = 'sent', sent_at = ?, last_error = ''
		WHERE id = ?`, nowUTC(), id)
	return err
}

func (s *Store) MarkDeliveryDisabled(ctx context.Context, id, userID, reason string) error {
	return withTx(ctx, s.DB, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE notification_deliveries SET status = 'disabled', last_error = ?
			WHERE id = ?`, truncate(reason, 500), id); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `
			UPDATE users SET bot_write_allowed = 0, updated_at = ? WHERE id = ?`,
			nowUTC(), userID)
		return err
	})
}

func (s *Store) RetryDelivery(ctx context.Context, id string, attempts int, reason string, retryAfter time.Duration) error {
	if retryAfter <= 0 {
		seconds := math.Min(3600, math.Pow(2, float64(attempts+1)))
		retryAfter = time.Duration(seconds)*time.Second + time.Duration(time.Now().UnixNano()%1000)*time.Millisecond
	}
	status := "failed"
	if attempts >= 9 {
		status = "disabled"
	}
	_, err := s.DB.ExecContext(ctx, `
		UPDATE notification_deliveries
		SET status = ?, next_attempt_at = ?, last_error = ?
		WHERE id = ?`,
		status, time.Now().UTC().Add(retryAfter).Format(time.RFC3339Nano), truncate(reason, 500), id)
	return err
}

func (s *Store) DeferDelivery(ctx context.Context, id string, until time.Time) error {
	_, err := s.DB.ExecContext(ctx, `
		UPDATE notification_deliveries
		SET status = 'pending', next_attempt_at = ?
		WHERE id = ?`, until.UTC().Format(time.RFC3339Nano), id)
	return err
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
