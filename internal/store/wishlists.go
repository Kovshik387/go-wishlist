package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/example/wishtrack/internal/platform"
)

func (s *Store) ListWishlists(ctx context.Context, ownerID string) ([]Wishlist, error) {
	rows, err := s.DB.QueryContext(ctx, wishlistSelect+`
		WHERE w.owner_id = ? AND w.deleted_at IS NULL
		ORDER BY w.created_at DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWishlists(rows, ownerID)
}

func (s *Store) ListSavedWishlists(ctx context.Context, userID string) ([]Wishlist, error) {
	rows, err := s.DB.QueryContext(ctx, wishlistSelect+`
		JOIN wishlist_access wa ON wa.wishlist_id = w.id
		WHERE wa.user_id = ? AND w.owner_id != ? AND w.deleted_at IS NULL
		ORDER BY wa.granted_at DESC`, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWishlists(rows, userID)
}

func (s *Store) ForgetWishlist(ctx context.Context, wishlistID, userID string) error {
	_, err := s.DB.ExecContext(ctx, `
		DELETE FROM wishlist_access
		WHERE wishlist_id = ? AND user_id = ?
		  AND wishlist_id IN (
		    SELECT id FROM wishlists WHERE owner_id != ?
		  )`, wishlistID, userID, userID)
	return err
}

func (s *Store) ListVisibleWishlistsByOwner(ctx context.Context, ownerID, viewerID string) ([]Wishlist, error) {
	rows, err := s.DB.QueryContext(ctx, wishlistSelect+`
		WHERE w.owner_id = ? AND w.deleted_at IS NULL
		  AND (
		    w.visibility = 'public'
		    OR (w.visibility = 'link' AND EXISTS (
		      SELECT 1 FROM wishlist_access wa
		      WHERE wa.wishlist_id = w.id AND wa.user_id = ?
		    ))
		  )
		ORDER BY w.created_at DESC`, ownerID, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWishlists(rows, viewerID)
}

func (s *Store) CreateWishlist(ctx context.Context, ownerID string, input WishlistInput) (Wishlist, error) {
	normalizeWishlistInput(&input)
	if err := validateWishlistInput(input); err != nil {
		return Wishlist{}, err
	}
	id := platform.UUID()
	token, err := platform.RandomToken(24)
	if err != nil {
		return Wishlist{}, err
	}
	now := nowUTC()
	err = withTx(ctx, s.DB, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO wishlists (
				id, owner_id, title, description, emoji, cover_url, occasion, event_date,
				visibility, allow_reservations, owner_sees_reservations, public_token,
				created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''), ?, ?, ?, ?, ?, ?)`,
			id, ownerID, input.Title, input.Description, input.Emoji, input.CoverURL,
			input.Occasion, input.EventDate, input.Visibility, boolInt(*input.AllowReservations),
			boolInt(*input.OwnerSeesReservations), token, now, now)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO activity_events(id, actor_id, event_type, entity_type, entity_id, created_at)
			VALUES (?, ?, 'wishlist.created', 'wishlist', ?, ?)`,
			platform.UUID(), ownerID, id, now)
		return err
	})
	if err != nil {
		return Wishlist{}, err
	}
	return s.GetWishlist(ctx, id, ownerID)
}

func (s *Store) GetWishlist(ctx context.Context, id, viewerID string) (Wishlist, error) {
	list, err := scanWishlist(s.DB.QueryRowContext(ctx, wishlistSelect+`
		WHERE w.id = ? AND w.deleted_at IS NULL`, id))
	if err != nil {
		return Wishlist{}, err
	}
	allowed, err := s.canViewWishlist(ctx, list, viewerID)
	if err != nil {
		return Wishlist{}, err
	}
	if !allowed {
		return Wishlist{}, ErrForbidden
	}
	if list.OwnerID != viewerID {
		list.PublicToken = ""
	}
	list.Wishes, err = s.listWishes(ctx, list, viewerID)
	return list, err
}

func (s *Store) PublicWishlist(ctx context.Context, token, viewerID string) (Wishlist, error) {
	list, err := scanWishlist(s.DB.QueryRowContext(ctx, wishlistSelect+`
		WHERE w.public_token = ? AND w.deleted_at IS NULL`, token))
	if err != nil {
		return Wishlist{}, err
	}
	if list.Visibility == "private" && list.OwnerID != viewerID {
		return Wishlist{}, ErrNotFound
	}
	if viewerID != "" && list.OwnerID != viewerID {
		_, err = s.DB.ExecContext(ctx, `
			INSERT INTO wishlist_access(wishlist_id, user_id, granted_at)
			VALUES (?, ?, ?)
			ON CONFLICT(wishlist_id, user_id) DO NOTHING`,
			list.ID, viewerID, nowUTC())
		if err != nil {
			return Wishlist{}, err
		}
	}
	if list.OwnerID != viewerID {
		list.PublicToken = ""
	}
	list.Wishes, err = s.listWishes(ctx, list, viewerID)
	return list, err
}

func (s *Store) UpdateWishlist(ctx context.Context, id, ownerID string, input WishlistInput) (Wishlist, error) {
	normalizeWishlistInput(&input)
	if err := validateWishlistInput(input); err != nil {
		return Wishlist{}, err
	}
	if input.Version <= 0 {
		return Wishlist{}, fmt.Errorf("version is required")
	}
	result, err := s.DB.ExecContext(ctx, `
		UPDATE wishlists SET
			title = ?, description = ?, emoji = ?, cover_url = ?, occasion = ?,
			event_date = NULLIF(?, ''), visibility = ?, allow_reservations = ?,
			owner_sees_reservations = ?, version = version + 1, updated_at = ?
		WHERE id = ? AND owner_id = ? AND version = ? AND deleted_at IS NULL`,
		input.Title, input.Description, input.Emoji, input.CoverURL, input.Occasion,
		input.EventDate, input.Visibility, boolInt(*input.AllowReservations),
		boolInt(*input.OwnerSeesReservations), nowUTC(), id, ownerID, input.Version)
	if err != nil {
		return Wishlist{}, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		var exists int
		_ = s.DB.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM wishlists WHERE id = ? AND owner_id = ? AND deleted_at IS NULL`,
			id, ownerID).Scan(&exists)
		if exists > 0 {
			return Wishlist{}, ErrVersionConflict
		}
		return Wishlist{}, ErrNotFound
	}
	return s.GetWishlist(ctx, id, ownerID)
}

func (s *Store) DeleteWishlist(ctx context.Context, id, ownerID string) error {
	now := nowUTC()
	return withTx(ctx, s.DB, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
			UPDATE wishlists SET deleted_at = ?, updated_at = ?, version = version + 1
			WHERE id = ? AND owner_id = ? AND deleted_at IS NULL`, now, now, id, ownerID)
		if err != nil {
			return err
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return ErrNotFound
		}
		if _, err = tx.ExecContext(ctx, `
			UPDATE wishes SET deleted_at = ?, updated_at = ?, version = version + 1
			WHERE wishlist_id = ? AND deleted_at IS NULL`, now, now, id); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
			UPDATE reservations SET status = 'archived', updated_at = ?
			WHERE status = 'active' AND wish_id IN (SELECT id FROM wishes WHERE wishlist_id = ?)`,
			now, id)
		return err
	})
}

func (s *Store) RotateWishlistLink(ctx context.Context, id, ownerID string) (Wishlist, error) {
	token, err := platform.RandomToken(24)
	if err != nil {
		return Wishlist{}, err
	}
	err = withTx(ctx, s.DB, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
			UPDATE wishlists SET public_token = ?, version = version + 1, updated_at = ?
			WHERE id = ? AND owner_id = ? AND deleted_at IS NULL`,
			token, nowUTC(), id, ownerID)
		if err != nil {
			return err
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return ErrNotFound
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM wishlist_access WHERE wishlist_id = ?`, id)
		return err
	})
	if err != nil {
		return Wishlist{}, err
	}
	return s.GetWishlist(ctx, id, ownerID)
}

func (s *Store) canViewWishlist(ctx context.Context, list Wishlist, viewerID string) (bool, error) {
	if list.OwnerID == viewerID || list.Visibility == "public" {
		return true, nil
	}
	if list.Visibility == "private" || viewerID == "" {
		return false, nil
	}
	var access int
	err := s.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM wishlist_access WHERE wishlist_id = ? AND user_id = ?`,
		list.ID, viewerID).Scan(&access)
	return access > 0, err
}

func normalizeWishlistInput(input *WishlistInput) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Emoji = strings.TrimSpace(input.Emoji)
	input.CoverURL = strings.TrimSpace(input.CoverURL)
	input.Occasion = strings.TrimSpace(input.Occasion)
	input.Visibility = strings.TrimSpace(input.Visibility)
	if input.Emoji == "" {
		input.Emoji = "🎁"
	}
	if input.Occasion == "" {
		input.Occasion = "other"
	}
	if input.Visibility == "" {
		input.Visibility = "public"
	}
	if input.AllowReservations == nil {
		value := true
		input.AllowReservations = &value
	}
	if input.OwnerSeesReservations == nil {
		value := false
		input.OwnerSeesReservations = &value
	}
}

func validateWishlistInput(input WishlistInput) error {
	if len([]rune(input.Title)) < 1 || len([]rune(input.Title)) > 80 {
		return fmt.Errorf("title must contain 1–80 characters")
	}
	if len([]rune(input.Description)) > 500 {
		return fmt.Errorf("description is too long")
	}
	switch input.Occasion {
	case "birthday", "wedding", "new_year", "housewarming", "other":
	default:
		return fmt.Errorf("invalid occasion")
	}
	switch input.Visibility {
	case "public", "link", "private":
	default:
		return fmt.Errorf("invalid visibility")
	}
	return nil
}

const wishlistSelect = `SELECT
	w.id, w.owner_id, w.title, w.description, w.emoji, w.cover_url, w.occasion,
	COALESCE(w.event_date, ''), w.visibility, w.allow_reservations,
	w.owner_sees_reservations, w.public_token, w.version, w.created_at, w.updated_at,
	(SELECT COUNT(*) FROM wishes wi WHERE wi.wishlist_id = w.id AND wi.deleted_at IS NULL),
	u.id, u.username, u.display_name, u.avatar_url, u.language_code, u.timezone, u.created_at
	FROM wishlists w
	JOIN users u ON u.id = w.owner_id`

func scanWishlist(row rowScanner) (Wishlist, error) {
	var list Wishlist
	var allow, ownerSees int
	var owner User
	err := row.Scan(
		&list.ID, &list.OwnerID, &list.Title, &list.Description, &list.Emoji, &list.CoverURL,
		&list.Occasion, &list.EventDate, &list.Visibility, &allow, &ownerSees,
		&list.PublicToken, &list.Version, &list.CreatedAt, &list.UpdatedAt, &list.WishCount,
		&owner.ID, &owner.Username, &owner.DisplayName, &owner.AvatarURL, &owner.LanguageCode,
		&owner.Timezone, &owner.CreatedAt,
	)
	if err != nil {
		return Wishlist{}, isNoRows(err)
	}
	list.AllowReservations = allow != 0
	list.OwnerSeesReservations = ownerSees != 0
	list.Owner = &owner
	return list, nil
}

func scanWishlists(rows *sql.Rows, ownerID string) ([]Wishlist, error) {
	lists := make([]Wishlist, 0)
	for rows.Next() {
		list, err := scanWishlist(rows)
		if err != nil {
			return nil, err
		}
		if list.OwnerID != ownerID {
			list.PublicToken = ""
		}
		lists = append(lists, list)
	}
	return lists, rows.Err()
}

func isConstraint(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "constraint failed") ||
		strings.Contains(err.Error(), "UNIQUE constraint"))
}

var _ = errors.Is
