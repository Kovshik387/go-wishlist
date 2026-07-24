package httpapi

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "image/gif"

	"github.com/example/wishtrack/internal/imagecache"
	"github.com/example/wishtrack/internal/platform"
)

const maxUploadBytes = 6 << 20

func (s *Server) uploadMedia(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "FILE_REQUIRED", "Выберите изображение")
		return
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
	if err != nil || len(body) > maxUploadBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "Изображение должно быть не больше 6 МБ")
		return
	}
	contentType, extension := imagecache.DetectImage(body)
	if contentType == "" {
		writeError(w, http.StatusUnsupportedMediaType, "UNSUPPORTED_IMAGE", "Поддерживаются JPEG, PNG, WebP, GIF и AVIF")
		return
	}

	var output bytes.Buffer
	if contentType == "image/jpeg" || contentType == "image/png" {
		cfg, _, decodeErr := image.DecodeConfig(bytes.NewReader(body))
		if decodeErr != nil || cfg.Width < 1 || cfg.Height < 1 || int64(cfg.Width)*int64(cfg.Height) > 24_000_000 {
			writeError(w, http.StatusUnprocessableEntity, "INVALID_IMAGE", "Не удалось обработать изображение")
			return
		}
		decoded, _, decodeErr := image.Decode(bytes.NewReader(body))
		if decodeErr != nil {
			writeError(w, http.StatusUnprocessableEntity, "INVALID_IMAGE", "Не удалось обработать изображение")
			return
		}
		if contentType == "image/png" {
			err = png.Encode(&output, decoded)
		} else {
			err = jpeg.Encode(&output, decoded, &jpeg.Options{Quality: 88})
		}
	} else {
		_, err = output.Write(body)
	}
	if err != nil {
		writeStoreError(w, err)
		return
	}

	filename := platform.UUID() + extension
	relativePath := filepath.Join(user.ID, filename)
	targetDir := filepath.Join(s.Config.MediaDir, user.ID)
	if err := os.MkdirAll(targetDir, 0o750); err != nil {
		writeStoreError(w, err)
		return
	}
	target := filepath.Join(targetDir, filename)
	outputFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if _, err = io.Copy(outputFile, &output); err != nil {
		_ = outputFile.Close()
		_ = os.Remove(target)
		writeStoreError(w, err)
		return
	}
	if err = outputFile.Close(); err != nil {
		_ = os.Remove(target)
		writeStoreError(w, err)
		return
	}
	publicURL := "/media/" + strings.ReplaceAll(filepath.ToSlash(relativePath), " ", "%20")
	media, err := s.Store.CreateMedia(r.Context(), user.ID, relativePath, publicURL, contentType, int64(output.Len()))
	if err != nil {
		_ = os.Remove(target)
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, media)
}

var _ = fmt.Sprintf
