package imagecache

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/example/wishtrack/internal/preview"
)

const DefaultMaxBytes = 6 << 20

type Asset struct {
	Body      []byte
	MimeType  string
	Extension string
}

type Fetcher struct {
	Resolver   preview.Resolver
	HTTPClient *http.Client
	Timeout    time.Duration
	MaxBytes   int64
}

func (f Fetcher) Fetch(ctx context.Context, rawURL, referer string) (Asset, error) {
	resolver := f.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	parsed, err := preview.ValidatePublicURL(ctx, resolver, rawURL)
	if err != nil {
		return Asset{}, err
	}
	timeout := f.Timeout
	if timeout <= 0 {
		timeout = 7 * time.Second
	}
	maxBytes := f.MaxBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	client := f.HTTPClient
	closeClient := func() {}
	if client == nil {
		client, closeClient = preview.NewSafeHTTPClient(resolver, timeout, 3)
	}
	defer closeClient()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return Asset{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; WishTrack/1.0)")
	req.Header.Set("Accept", "image/avif,image/webp,image/png,image/jpeg,image/gif;q=0.8,*/*;q=0.2")
	if strings.HasPrefix(referer, "http://") || strings.HasPrefix(referer, "https://") {
		req.Header.Set("Referer", referer)
	}
	resp, err := client.Do(req)
	if err != nil {
		return Asset{}, fmt.Errorf("download image: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Asset{}, fmt.Errorf("image returned HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > maxBytes {
		return Asset{}, errors.New("image is too large")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return Asset{}, err
	}
	if int64(len(body)) > maxBytes {
		return Asset{}, errors.New("image is too large")
	}
	mimeType, extension := detectImage(body)
	if mimeType == "" {
		return Asset{}, errors.New("remote file is not a supported image")
	}
	return Asset{Body: body, MimeType: mimeType, Extension: extension}, nil
}

func detectImage(body []byte) (string, string) {
	switch {
	case len(body) >= 3 && bytes.Equal(body[:3], []byte{0xff, 0xd8, 0xff}):
		return "image/jpeg", ".jpg"
	case len(body) >= 8 && bytes.Equal(body[:8], []byte("\x89PNG\r\n\x1a\n")):
		return "image/png", ".png"
	case len(body) >= 6 && (bytes.Equal(body[:6], []byte("GIF87a")) || bytes.Equal(body[:6], []byte("GIF89a"))):
		return "image/gif", ".gif"
	case len(body) >= 12 && bytes.Equal(body[:4], []byte("RIFF")) && bytes.Equal(body[8:12], []byte("WEBP")):
		return "image/webp", ".webp"
	case len(body) >= 12 && bytes.Equal(body[4:8], []byte("ftyp")) &&
		(bytes.Equal(body[8:12], []byte("avif")) || bytes.Equal(body[8:12], []byte("avis"))):
		return "image/avif", ".avif"
	default:
		return "", ""
	}
}
