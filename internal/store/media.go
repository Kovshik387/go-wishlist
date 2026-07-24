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
