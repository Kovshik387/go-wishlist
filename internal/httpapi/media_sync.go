package httpapi

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/example/wishtrack/internal/imagecache"
	"github.com/example/wishtrack/internal/platform"
	"github.com/example/wishtrack/internal/store"
)

const maxImagesPerSync = 12

type imageSyncResult struct {
	Attempted int `json:"attempted"`
	Synced    int `json:"synced"`
	Failed    int `json:"failed"`
}

type fetchedWishImage struct {
	item  store.RemoteWishImage
	asset imagecache.Asset
	err   error
}

func (s *Server) syncMedia(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	result, err := s.syncRemoteWishImages(ctx, user.ID)
	if err != nil {
		s.Logger.Warn("sync remote wish images", "user_id", user.ID, "error", err)
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) syncRemoteWishImages(ctx context.Context, ownerID string) (imageSyncResult, error) {
	items, err := s.Store.ListRemoteWishImages(ctx, ownerID, maxImagesPerSync)
	if err != nil {
		return imageSyncResult{}, err
	}
	result := imageSyncResult{Attempted: len(items)}
	if len(items) == 0 {
		return result, nil
	}

	fetched := make(chan fetchedWishImage, len(items))
	parallel := make(chan struct{}, 4)
	for _, item := range items {
		item := item
		go func() {
			select {
			case parallel <- struct{}{}:
				defer func() { <-parallel }()
			case <-ctx.Done():
				fetched <- fetchedWishImage{item: item, err: ctx.Err()}
				return
			}
			asset, fetchErr := s.Images.Fetch(ctx, item.ImageURL, item.ProductURL)
			fetched <- fetchedWishImage{item: item, asset: asset, err: fetchErr}
		}()
	}

	for range items {
		downloaded := <-fetched
		if downloaded.err != nil {
			result.Failed++
			s.Logger.Debug("remote wish image unavailable",
				"wish_id", downloaded.item.WishID, "error", downloaded.err)
			continue
		}
		publicURL, persistErr := s.persistImage(ctx, ownerID, downloaded.asset)
		if persistErr != nil {
			result.Failed++
			s.Logger.Warn("persist remote wish image",
				"wish_id", downloaded.item.WishID, "error", persistErr)
			continue
		}
		replaced, replaceErr := s.Store.ReplaceRemoteWishImage(
			ctx, ownerID, downloaded.item.WishID, downloaded.item.ImageURL, publicURL,
		)
		if replaceErr != nil || !replaced {
			result.Failed++
			if replaceErr != nil {
				s.Logger.Warn("attach cached wish image",
					"wish_id", downloaded.item.WishID, "error", replaceErr)
			}
			continue
		}
		result.Synced++
	}
	return result, nil
}

func (s *Server) cacheRemoteImage(ctx context.Context, ownerID, rawURL, referer string) (string, error) {
	if !isRemoteImageURL(rawURL) {
		return rawURL, nil
	}
	asset, err := s.Images.Fetch(ctx, rawURL, referer)
	if err != nil {
		return "", err
	}
	return s.persistImage(ctx, ownerID, asset)
}

func (s *Server) cacheTelegramAvatar(ctx context.Context, ownerID, rawURL string) (string, error) {
	asset, err := s.Images.Fetch(ctx, rawURL, "")
	if err != nil {
		return "", err
	}
	targetDir := filepath.Join(s.Config.MediaDir, ownerID, "avatar")
	if err = os.MkdirAll(targetDir, 0o750); err != nil {
		return "", err
	}
	filename := "current" + asset.Extension
	target := filepath.Join(targetDir, filename)
	output, err := os.CreateTemp(targetDir, ".avatar-*")
	if err != nil {
		return "", err
	}
	temporaryPath := output.Name()
	defer os.Remove(temporaryPath)
	if err = output.Chmod(0o640); err != nil {
		_ = output.Close()
		return "", err
	}
	if _, err = io.Copy(output, bytes.NewReader(asset.Body)); err != nil {
		_ = output.Close()
		return "", err
	}
	if err = output.Close(); err != nil {
		return "", err
	}
	if err = os.Rename(temporaryPath, target); err != nil {
		return "", err
	}
	relativePath := filepath.Join(ownerID, "avatar", filename)
	return "/media/" + filepath.ToSlash(relativePath) + "?v=" +
		strconv.FormatInt(time.Now().Unix(), 10), nil
}

func (s *Server) persistImage(ctx context.Context, ownerID string, asset imagecache.Asset) (string, error) {
	filename := platform.UUID() + asset.Extension
	relativePath := filepath.Join(ownerID, "remote", filename)
	targetDir := filepath.Join(s.Config.MediaDir, ownerID, "remote")
	if err := os.MkdirAll(targetDir, 0o750); err != nil {
		return "", err
	}
	target := filepath.Join(targetDir, filename)
	output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return "", err
	}
	if _, err = io.Copy(output, bytes.NewReader(asset.Body)); err != nil {
		_ = output.Close()
		_ = os.Remove(target)
		return "", err
	}
	if err = output.Close(); err != nil {
		_ = os.Remove(target)
		return "", err
	}
	publicURL := "/media/" + strings.ReplaceAll(filepath.ToSlash(relativePath), " ", "%20")
	if _, err = s.Store.CreateMedia(
		ctx, ownerID, relativePath, publicURL, asset.MimeType, int64(len(asset.Body)),
	); err != nil {
		_ = os.Remove(target)
		return "", err
	}
	return publicURL, nil
}

func isRemoteImageURL(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}
