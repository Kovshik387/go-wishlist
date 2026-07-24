package preview

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var ErrUnsafeURL = errors.New("URL points to a non-public network")

type Resolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

type Fetcher struct {
	Resolver Resolver
	Timeout  time.Duration
	MaxBytes int64
}

type Result struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageURL    string `json:"imageUrl"`
	PriceMinor  *int64 `json:"priceMinor,omitempty"`
	Currency    string `json:"currency"`
	StoreDomain string `json:"storeDomain"`
}

func (f Fetcher) Fetch(ctx context.Context, rawURL string) (Result, error) {
	resolver := f.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	parsed, err := ValidatePublicURL(ctx, resolver, rawURL)
	if err != nil {
		return Result{}, err
	}
	timeout := f.Timeout
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	maxBytes := f.MaxBytes
	if maxBytes <= 0 {
		maxBytes = 2 << 20
	}
	client, closeClient := NewSafeHTTPClient(resolver, timeout, 3)
	defer closeClient()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("User-Agent", "WishTrack-LinkPreview/1.0 (+https://wishtrack.app)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("fetch product page: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("product page returned HTTP %d", resp.StatusCode)
	}
	contentType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if contentType != "text/html" && contentType != "application/xhtml+xml" {
		return Result{}, fmt.Errorf("unsupported Content-Type %q", contentType)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return Result{}, err
	}
	if int64(len(body)) > maxBytes {
		return Result{}, errors.New("product page is too large")
	}
	finalURL := resp.Request.URL
	result, err := parseHTML(body, finalURL)
	if err != nil {
		return Result{}, err
	}
	result.URL = finalURL.String()
	result.StoreDomain = finalURL.Hostname()
	return result, nil
}

func ValidatePublicURL(ctx context.Context, resolver Resolver, rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Hostname() == "" {
		return nil, errors.New("invalid URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("only http and https URLs are supported")
	}
	if parsed.User != nil {
		return nil, errors.New("URLs with credentials are not supported")
	}
	if strings.EqualFold(parsed.Hostname(), "localhost") ||
		strings.HasSuffix(strings.ToLower(parsed.Hostname()), ".localhost") {
		return nil, ErrUnsafeURL
	}
	addresses, err := resolver.LookupIPAddr(ctx, parsed.Hostname())
	if err != nil {
		return nil, fmt.Errorf("resolve host: %w", err)
	}
	if len(addresses) == 0 {
		return nil, errors.New("host has no IP addresses")
	}
	for _, address := range addresses {
		if !IsPublicIP(address.IP) {
			return nil, ErrUnsafeURL
		}
	}
	return parsed, nil
}

func NewSafeHTTPClient(resolver Resolver, timeout time.Duration, maxRedirects int) (*http.Client, func()) {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	if maxRedirects <= 0 {
		maxRedirects = 3
	}
	dialer := &net.Dialer{Timeout: 4 * time.Second, KeepAlive: 15 * time.Second}
	transport := &http.Transport{
		Proxy:                 nil,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   4 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		DialContext: func(dialCtx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			addresses, err := resolver.LookupIPAddr(dialCtx, host)
			if err != nil {
				return nil, err
			}
			for _, candidate := range addresses {
				if IsPublicIP(candidate.IP) {
					return dialer.DialContext(dialCtx, network, net.JoinHostPort(candidate.IP.String(), port))
				}
			}
			return nil, ErrUnsafeURL
		},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return errors.New("too many redirects")
			}
			_, err := ValidatePublicURL(req.Context(), resolver, req.URL.String())
			return err
		},
	}
	return client, transport.CloseIdleConnections
}

var blockedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("::/128"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("ff00::/8"),
}

func IsPublicIP(ip net.IP) bool {
	address, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	address = address.Unmap()
	if !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() ||
		address.IsLinkLocalUnicast() || address.IsMulticast() || address.IsUnspecified() {
		return false
	}
	for _, prefix := range blockedPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

func parseHTML(body []byte, base *url.URL) (Result, error) {
	root, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return Result{}, errors.New("cannot parse product page")
	}
	meta := map[string]string{}
	var pageTitle string
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "meta" {
			var key, content string
			for _, attr := range node.Attr {
				switch strings.ToLower(attr.Key) {
				case "property", "name", "itemprop":
					key = strings.ToLower(attr.Val)
				case "content":
					content = strings.TrimSpace(attr.Val)
				}
			}
			if key != "" && content != "" {
				if _, exists := meta[key]; !exists {
					meta[key] = content
				}
			}
		}
		if node.Type == html.ElementNode && node.Data == "title" && node.FirstChild != nil {
			pageTitle = strings.TrimSpace(node.FirstChild.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(root)
	result := Result{
		Title:       first(meta["og:title"], meta["twitter:title"], pageTitle),
		Description: first(meta["og:description"], meta["description"], meta["twitter:description"]),
		ImageURL:    first(meta["og:image:secure_url"], meta["og:image"], meta["twitter:image"]),
		Currency:    strings.ToUpper(first(meta["product:price:currency"], meta["og:price:currency"], "RUB")),
	}
	if result.ImageURL != "" {
		if imageURL, err := base.Parse(result.ImageURL); err == nil &&
			(imageURL.Scheme == "http" || imageURL.Scheme == "https") {
			result.ImageURL = imageURL.String()
		} else {
			result.ImageURL = ""
		}
	}
	rawPrice := first(meta["product:price:amount"], meta["og:price:amount"], meta["price"])
	if price := parsePrice(rawPrice); price != nil {
		result.PriceMinor = price
	}
	if len([]rune(result.Title)) > 200 {
		result.Title = string([]rune(result.Title)[:200])
	}
	if len([]rune(result.Description)) > 500 {
		result.Description = string([]rune(result.Description)[:500])
	}
	return result, nil
}

var nonPrice = regexp.MustCompile(`[^\d.,]`)

func parsePrice(value string) *int64 {
	value = strings.TrimSpace(nonPrice.ReplaceAllString(value, ""))
	if value == "" {
		return nil
	}
	if strings.Contains(value, ",") && !strings.Contains(value, ".") {
		value = strings.ReplaceAll(value, ",", ".")
	} else {
		value = strings.ReplaceAll(value, ",", "")
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed < 0 {
		return nil
	}
	minor := int64(parsed*100 + 0.5)
	return &minor
}

func first(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
