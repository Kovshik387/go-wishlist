package store

import (
	"context"

	"github.com/example/wishtrack/internal/platform"
)

type MediaObject struct {
	ID        string `json:"id"`
	PublicURL string `json:"publicUrl"`
	MimeType  string `json:"mimeType"`
	SizeBytes int64  `json:"sizeBytes"`
}

type RemoteWishImage struct {
	WishID     string
	ImageURL   string
	ProductURL string
}

func (s *Store) CreateMedia(ctx context.Context, ownerID, storagePath, publicURL, mimeType string, size int64) (MediaObject, error) {
	media := MediaObject{
		ID: platform.UUID(), PublicURL: publicURL, MimeType: mimeType, SizeBytes: size,
	}
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO media_objects(
			id, owner_id, storage_path, public_url, mime_type, size_bytes, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		media.ID, ownerID, storagePath, publicURL, mimeType, size, nowUTC())
	return media, err
}

func (s *Store) ListRemoteWishImages(ctx context.Context, ownerID string, limit int) ([]RemoteWishImage, error) {
	if limit <= 0 {
		limit = 12
	}
	rows, err := s.DB.QueryContext(ctx, `
		SELECT wi.id, wi.image_url, wi.product_url
		FROM wishes wi
		JOIN wishlists w ON w.id = wi.wishlist_id
		WHERE w.owner_id = ? AND w.deleted_at IS NULL AND wi.deleted_at IS NULL
			AND (LOWER(wi.image_url) LIKE 'http://%' OR LOWER(wi.image_url) LIKE 'https://%')
		ORDER BY wi.updated_at ASC
		LIMIT ?`, ownerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]RemoteWishImage, 0)
	for rows.Next() {
		var item RemoteWishImage
		if err := rows.Scan(&item.WishID, &item.ImageURL, &item.ProductURL); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) ReplaceRemoteWishImage(ctx context.Context, ownerID, wishID, oldURL, localURL string) (bool, error) {
	result, err := s.DB.ExecContext(ctx, `
		UPDATE wishes
		SET image_url = ?, updated_at = ?
		WHERE id = ? AND image_url = ? AND deleted_at IS NULL
			AND wishlist_id IN (
				SELECT id FROM wishlists WHERE owner_id = ? AND deleted_at IS NULL
			)`, localURL, nowUTC(), wishID, oldURL, ownerID)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected > 0, err
}
