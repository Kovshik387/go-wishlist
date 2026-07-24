package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/example/wishtrack/internal/platform"
)

func (s *Store) CreateWish(ctx context.Context, ownerID, wishlistID string, input WishInput, digestWindow time.Duration) (Wish, error) {
	normalizeWishInput(&input)
	if err := validateWishInput(input); err != nil {
		return Wish{}, err
	}
	attributes, _ := json.Marshal(input.Attributes)
	wishID := platform.UUID()
	eventID := platform.UUID()
	now := time.Now().UTC()
	payload, _ := json.Marshal(map[string]any{
		"wishId": wishID, "wishlistId": wishlistID, "authorId": ownerID,
	})
	err := withTx(ctx, s.DB, func(tx *sql.Tx) error {
		var visibility string
		err := tx.QueryRowContext(ctx, `
			SELECT visibility FROM wishlists
			WHERE id = ? AND owner_id = ? AND deleted_at IS NULL`,
			wishlistID, ownerID).Scan(&visibility)
		if err != nil {
			return isNoRows(err)
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO wishes (
				id, wishlist_id, product_url, title, description, image_url, price_minor,
				currency, priority, quantity, attributes_json, store_domain, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			wishID, wishlistID, input.ProductURL, input.Title, input.Description, input.ImageURL,
			input.PriceMinor, input.Currency, input.Priority, input.Quantity, string(attributes),
			input.StoreDomain, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO activity_events(id, actor_id, event_type, entity_type, entity_id, payload, created_at)
			VALUES (?, ?, 'wish.created', 'wish', ?, ?, ?)`,
			platform.UUID(), ownerID, wishID, string(payload), now.Format(time.RFC3339Nano)); err != nil {
			return err
		}
		if visibility == "public" {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO outbox_events(
					id, event_type, aggregate_id, payload, available_at, created_at
				) VALUES (?, 'wish.created', ?, ?, ?, ?)`,
				eventID, wishID, string(payload), now.Add(digestWindow).Format(time.RFC3339Nano),
				now.Format(time.RFC3339Nano))
		}
		return err
	})
	if err != nil {
		return Wish{}, err
	}
	list, err := s.GetWishlist(ctx, wishlistID, ownerID)
	if err != nil {
		return Wish{}, err
	}
	for _, wish := range list.Wishes {
		if wish.ID == wishID {
			return wish, nil
		}
	}
	return Wish{}, ErrNotFound
}

func (s *Store) GetWish(ctx context.Context, id, viewerID string) (Wish, error) {
	list, err := scanWishlist(s.DB.QueryRowContext(ctx, wishlistSelect+`
		JOIN wishes target ON target.wishlist_id = w.id
		WHERE target.id = ? AND target.deleted_at IS NULL AND w.deleted_at IS NULL`, id))
	if err != nil {
		return Wish{}, err
	}
	allowed, err := s.canViewWishlist(ctx, list, viewerID)
	if err != nil {
		return Wish{}, err
	}
	if !allowed {
		return Wish{}, ErrForbidden
	}
	wishes, err := s.queryWishes(ctx, `wi.id = ?`, []any{id}, list, viewerID)
	if err != nil {
		return Wish{}, err
	}
	if len(wishes) == 0 {
		return Wish{}, ErrNotFound
	}
	return wishes[0], nil
}

func (s *Store) UpdateWish(ctx context.Context, id, ownerID string, input WishInput) (Wish, error) {
	normalizeWishInput(&input)
	if err := validateWishInput(input); err != nil {
		return Wish{}, err
	}
	if input.Version <= 0 {
		return Wish{}, fmt.Errorf("version is required")
	}
	attributes, _ := json.Marshal(input.Attributes)
	var wishlistID string
	err := withTx(ctx, s.DB, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `
			SELECT wi.wishlist_id FROM wishes wi
			JOIN wishlists w ON w.id = wi.wishlist_id
			WHERE wi.id = ? AND w.owner_id = ? AND wi.deleted_at IS NULL AND w.deleted_at IS NULL`,
			id, ownerID).Scan(&wishlistID)
		if err != nil {
			return isNoRows(err)
		}
		result, err := tx.ExecContext(ctx, `
			UPDATE wishes SET product_url = ?, title = ?, description = ?, image_url = ?,
				price_minor = ?, currency = ?, priority = ?, quantity = ?, attributes_json = ?,
				store_domain = ?, version = version + 1, updated_at = ?
			WHERE id = ? AND version = ? AND deleted_at IS NULL`,
			input.ProductURL, input.Title, input.Description, input.ImageURL, input.PriceMinor,
			input.Currency, input.Priority, input.Quantity, string(attributes), input.StoreDomain,
			nowUTC(), id, input.Version)
		if err != nil {
			return err
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return ErrVersionConflict
		}
		return nil
	})
	if err != nil {
		return Wish{}, err
	}
	return s.GetWish(ctx, id, ownerID)
}

func (s *Store) DeleteWish(ctx context.Context, id, ownerID string) error {
	now := nowUTC()
	return withTx(ctx, s.DB, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
			UPDATE wishes SET deleted_at = ?, updated_at = ?, version = version + 1
			WHERE id = ? AND deleted_at IS NULL AND wishlist_id IN (
				SELECT id FROM wishlists WHERE owner_id = ? AND deleted_at IS NULL
			)`, now, now, id, ownerID)
		if err != nil {
			return err
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return ErrNotFound
		}
		if _, err = tx.ExecContext(ctx, `
			UPDATE reservations SET status = 'archived', updated_at = ?
			WHERE wish_id = ? AND status = 'active'`, now, id); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO activity_events(id, actor_id, event_type, entity_type, entity_id, created_at)
			VALUES (?, ?, 'wish.deleted', 'wish', ?, ?)`,
			platform.UUID(), ownerID, id, now)
		return err
	})
}

func (s *Store) listWishes(ctx context.Context, list Wishlist, viewerID string) ([]Wish, error) {
	return s.queryWishes(ctx, `wi.wishlist_id = ?`, []any{list.ID}, list, viewerID)
}

func (s *Store) queryWishes(ctx context.Context, predicate string, args []any, list Wishlist, viewerID string) ([]Wish, error) {
	rows, err := s.DB.QueryContext(ctx, wishSelect+` WHERE `+predicate+`
		AND wi.deleted_at IS NULL ORDER BY wi.created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	wishes := make([]Wish, 0)
	for rows.Next() {
		wish, reservation, reserver, err := scanWish(rows)
		if err != nil {
			return nil, err
		}
		applyReservationVisibility(&wish, reservation, reserver, list, viewerID)
		wishes = append(wishes, wish)
	}
	return wishes, rows.Err()
}

type reservationView struct {
	ID           string
	ReservedByID string
}

func scanWish(row rowScanner) (Wish, reservationView, *User, error) {
	var wish Wish
	var price sql.NullInt64
	var attributes string
	var reservationID, reservedBy sql.NullString
	var reserverID, username, displayName, avatarURL sql.NullString
	err := row.Scan(
		&wish.ID, &wish.WishlistID, &wish.ProductURL, &wish.Title, &wish.Description,
		&wish.ImageURL, &price, &wish.Currency, &wish.Priority, &wish.Quantity,
		&attributes, &wish.StoreDomain, &wish.Version, &wish.CreatedAt, &wish.UpdatedAt,
		&reservationID, &reservedBy, &reserverID, &username, &displayName, &avatarURL,
	)
	if err != nil {
		return Wish{}, reservationView{}, nil, isNoRows(err)
	}
	if price.Valid {
		wish.PriceMinor = &price.Int64
	}
	wish.Attributes = map[string]string{}
	_ = json.Unmarshal([]byte(attributes), &wish.Attributes)
	reservation := reservationView{ID: reservationID.String, ReservedByID: reservedBy.String}
	var reserver *User
	if reserverID.Valid {
		reserver = &User{
			ID: reserverID.String, Username: username.String, DisplayName: displayName.String,
			AvatarURL: avatarURL.String,
		}
	}
	return wish, reservation, reserver, nil
}

func applyReservationVisibility(wish *Wish, reservation reservationView, reserver *User, list Wishlist, viewerID string) {
	if reservation.ID == "" {
		return
	}
	isOwner := list.OwnerID == viewerID
	if isOwner && !list.OwnerSeesReservations {
		return
	}
	wish.IsReserved = true
	wish.ReservedByMe = reservation.ReservedByID == viewerID && viewerID != ""
	if wish.ReservedByMe || (isOwner && list.OwnerSeesReservations) {
		wish.ReservedBy = reserver
	}
}

func normalizeWishInput(input *WishInput) {
	input.ProductURL = strings.TrimSpace(input.ProductURL)
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.ImageURL = strings.TrimSpace(input.ImageURL)
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	input.Priority = strings.TrimSpace(input.Priority)
	input.StoreDomain = strings.TrimSpace(input.StoreDomain)
	if input.Currency == "" {
		input.Currency = "RUB"
	}
	if input.Priority == "" {
		input.Priority = "normal"
	}
	if input.Quantity == 0 {
		input.Quantity = 1
	}
	if input.Attributes == nil {
		input.Attributes = map[string]string{}
	}
	if input.StoreDomain == "" && input.ProductURL != "" {
		if parsed, err := url.Parse(input.ProductURL); err == nil {
			input.StoreDomain = parsed.Hostname()
		}
	}
}

func validateWishInput(input WishInput) error {
	if len([]rune(input.Title)) < 1 || len([]rune(input.Title)) > 160 {
		return fmt.Errorf("title must contain 1–160 characters")
	}
	if len([]rune(input.Description)) > 2000 {
		return fmt.Errorf("description is too long")
	}
	if input.ProductURL != "" {
		parsed, err := url.ParseRequestURI(input.ProductURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return fmt.Errorf("invalid product URL")
		}
	}
	if input.PriceMinor != nil && *input.PriceMinor < 0 {
		return fmt.Errorf("price cannot be negative")
	}
	if len(input.Currency) != 3 {
		return fmt.Errorf("currency must be an ISO 4217 code")
	}
	if input.Priority != "normal" && input.Priority != "high" {
		return fmt.Errorf("invalid priority")
	}
	if input.Quantity < 1 || input.Quantity > 99 {
		return fmt.Errorf("quantity must be between 1 and 99")
	}
	return nil
}

const wishSelect = `SELECT
	wi.id, wi.wishlist_id, wi.product_url, wi.title, wi.description, wi.image_url,
	wi.price_minor, wi.currency, wi.priority, wi.quantity, wi.attributes_json,
	wi.store_domain, wi.version, wi.created_at, wi.updated_at,
	r.id, r.reserved_by,
	ru.id, ru.username, ru.display_name, ru.avatar_url
	FROM wishes wi
	LEFT JOIN reservations r ON r.wish_id = wi.id AND r.status = 'active'
	LEFT JOIN users ru ON ru.id = r.reserved_by`
