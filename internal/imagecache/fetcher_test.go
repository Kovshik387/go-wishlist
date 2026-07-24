package imagecache

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/example/wishtrack/internal/preview"
)

type testResolver map[string]net.IP

func (r testResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	ip, ok := r[host]
	if !ok {
		return nil, errors.New("not found")
	}
	return []net.IPAddr{{IP: ip}}, nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestFetchAcceptsSupportedPublicImage(t *testing.T) {
	pngBody := append([]byte("\x89PNG\r\n\x1a\n"), []byte("image-data")...)
	fetcher := Fetcher{
		Resolver: testResolver{"images.example": net.ParseIP("93.184.216.34")},
		HTTPClient: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			if got := request.Header.Get("Referer"); got != "https://shop.example/product" {
				t.Fatalf("Referer = %q", got)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(string(pngBody))),
			}, nil
		})},
	}
	asset, err := fetcher.Fetch(
		context.Background(),
		"https://images.example/photo.png",
		"https://shop.example/product",
	)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if asset.MimeType != "image/png" || asset.Extension != ".png" {
		t.Fatalf("unexpected asset metadata: %#v", asset)
	}
}

func TestFetchRejectsPrivateNetwork(t *testing.T) {
	called := false
	fetcher := Fetcher{
		Resolver: testResolver{"private.example": net.ParseIP("192.168.1.10")},
		HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			called = true
			return nil, errors.New("must not be called")
		})},
	}
	_, err := fetcher.Fetch(context.Background(), "http://private.example/photo.jpg", "")
	if !errors.Is(err, preview.ErrUnsafeURL) {
		t.Fatalf("error = %v, want ErrUnsafeURL", err)
	}
	if called {
		t.Fatal("HTTP client was called for a private network URL")
	}
}
