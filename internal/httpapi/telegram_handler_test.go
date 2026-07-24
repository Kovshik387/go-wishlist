package httpapi

import "testing"

func TestWishlistStartParam(t *testing.T) {
	if !validWishlistStartParam("wishlist_abc-DEF_123") {
		t.Fatal("valid wishlist start parameter was rejected")
	}
	for _, value := range []string{
		"",
		"wishlist_",
		"profile_abc",
		"wishlist_bad value",
		"wishlist_../../secret",
	} {
		if validWishlistStartParam(value) {
			t.Fatalf("invalid start parameter %q was accepted", value)
		}
	}
}

func TestWebAppURL(t *testing.T) {
	got := webAppURL("https://wish.example:8443", "wishlist_abc-123")
	want := "https://wish.example:8443?tgWebAppStartParam=wishlist_abc-123"
	if got != want {
		t.Fatalf("webAppURL() = %q, want %q", got, want)
	}
}
