package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/example/wishtrack/internal/platform"
)

func (s *Store) Follow(ctx context.Context, followerID, followeeID string) error {
	if followerID == followeeID {
		return fmt.Errorf("cannot follow yourself")
	}
	return withTx(ctx, s.DB, func(tx *sql.Tx) error {
		var exists int
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM users WHERE id = ? AND deleted_at IS NULL`, followeeID).Scan(&exists); err != nil {
			return err
		}
		if exists == 0 {
			return ErrNotFound
		}
		now := nowUTC()
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO follows(follower_id, followee_id, created_at)
			VALUES (?, ?, ?)
			ON CONFLICT(follower_id, followee_id) DO NOTHING`,
			followerID, followeeID, now); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO follow_notification_settings(follower_id, followee_id, muted, updated_at)
			VALUES (?, ?, 0, ?)
			ON CONFLICT(follower_id, followee_id) DO NOTHING`,
			followerID, followeeID, now)
		return err
	})
}

func (s *Store) Unfollow(ctx context.Context, followerID, followeeID string) error {
	_, err := s.DB.ExecContext(ctx,
		`DELETE FROM follows WHERE follower_id = ? AND followee_id = ?`,
		followerID, followeeID)
	return err
}

func (s *Store) SetFollowMuted(ctx context.Context, followerID, followeeID string, muted bool) error {
	result, err := s.DB.ExecContext(ctx, `
		UPDATE follow_notification_settings SET muted = ?, updated_at = ?
		WHERE follower_id = ? AND followee_id = ?`,
		boolInt(muted), nowUTC(), followerID, followeeID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Feed(ctx context.Context, viewerID string, limit int) ([]Wish, error) {
	if limit < 1 || limit > 100 {
		limit = 30
	}
	rows, err := s.DB.QueryContext(ctx, `
		SELECT wi.id
		FROM wishes wi
		JOIN wishlists w ON w.id = wi.wishlist_id
		JOIN follows f ON f.followee_id = w.owner_id
		WHERE f.follower_id = ?
		  AND w.visibility = 'public'
		  AND w.deleted_at IS NULL
		  AND wi.deleted_at IS NULL
		ORDER BY wi.created_at DESC
		LIMIT ?`, viewerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	feed := make([]Wish, 0, len(ids))
	for _, id := range ids {
		wish, err := s.GetWish(ctx, id, viewerID)
		if err != nil {
			return nil, err
		}
		list, err := s.GetWishlist(ctx, wish.WishlistID, viewerID)
		if err != nil {
			return nil, err
		}
		list.Wishes = nil
		list.PublicToken = ""
		wish.Wishlist = &list
		wish.Author = list.Owner
		feed = append(feed, wish)
	}
	return feed, nil
}

func (s *Store) ReserveWish(ctx context.Context, wishID, userID string) (Reservation, error) {
	var reservation Reservation
	err := withTx(ctx, s.DB, func(tx *sql.Tx) error {
		var ownerID string
		var allow int
		err := tx.QueryRowContext(ctx, `
			SELECT w.owner_id, w.allow_reservations
			FROM wishes wi
			JOIN wishlists w ON w.id = wi.wishlist_id
			WHERE wi.id = ? AND wi.deleted_at IS NULL AND w.deleted_at IS NULL`,
			wishID).Scan(&ownerID, &allow)
		if err != nil {
			return isNoRows(err)
		}
		if ownerID == userID {
			return ErrOwnWish
		}
		if allow == 0 {
			return ErrReservationsOff
		}
		now := nowUTC()
		reservation = Reservation{
			ID: platform.UUID(), WishID: wishID, Status: "active", CreatedAt: now,
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO reservations(id, wish_id, reserved_by, status, created_at, updated_at)
			VALUES (?, ?, ?, 'active', ?, ?)`,
			reservation.ID, wishID, userID, now, now)
		if isConstraint(err) {
			return ErrAlreadyReserved
		}
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO activity_events(id, actor_id, event_type, entity_type, entity_id, created_at)
			VALUES (?, ?, 'reservation.created', 'wish', ?, ?)`,
			platform.UUID(), userID, wishID, now)
		return err
	})
	if err != nil {
		return Reservation{}, err
	}
	wish, err := s.GetWish(ctx, wishID, userID)
	if err != nil {
		return Reservation{}, err
	}
	reservation.Wish = wish
	return reservation, nil
}

func (s *Store) CancelReservation(ctx context.Context, wishID, userID string) error {
	now := nowUTC()
	return withTx(ctx, s.DB, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
			UPDATE reservations SET status = 'cancelled', cancelled_at = ?, updated_at = ?
			WHERE wish_id = ? AND reserved_by = ? AND status = 'active'`,
			now, now, wishID, userID)
		if err != nil {
			return err
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return ErrNotFound
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO activity_events(id, actor_id, event_type, entity_type, entity_id, created_at)
			VALUES (?, ?, 'reservation.cancelled', 'wish', ?, ?)`,
			platform.UUID(), userID, wishID, now)
		return err
	})
}

func (s *Store) Reservations(ctx context.Context, userID string) ([]Reservation, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, wish_id, status, created_at
		FROM reservations
		WHERE reserved_by = ? AND status = 'active'
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Reservation, 0)
	for rows.Next() {
		var item Reservation
		if err := rows.Scan(&item.ID, &item.WishID, &item.Status, &item.CreatedAt); err != nil {
			return nil, err
		}
		wish, err := s.GetWish(ctx, item.WishID, userID)
		if err != nil {
			if err == ErrForbidden || err == ErrNotFound {
				continue
			}
			return nil, err
		}
		item.Wish = wish
		items = append(items, item)
	}
	return items, rows.Err()
}
