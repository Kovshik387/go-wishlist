package notification

import (
	"testing"
	"time"

	"github.com/example/wishtrack/internal/store"
)

func TestQuietUntilOvernight(t *testing.T) {
	now := time.Date(2026, 7, 24, 21, 30, 0, 0, time.UTC) // 00:30 Moscow
	until, quiet := QuietUntil(now, "Europe/Moscow", store.NotificationPreferences{
		QuietHoursEnabled: true, QuietStart: "23:00", QuietEnd: "08:00",
	})
	if !quiet {
		t.Fatal("expected quiet hours")
	}
	want := time.Date(2026, 7, 25, 5, 0, 0, 0, time.UTC)
	if !until.Equal(want) {
		t.Fatalf("until = %v, want %v", until, want)
	}
}

func TestQuietUntilDaytime(t *testing.T) {
	now := time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
	_, quiet := QuietUntil(now, "Europe/Moscow", store.NotificationPreferences{
		QuietHoursEnabled: true, QuietStart: "23:00", QuietEnd: "08:00",
	})
	if quiet {
		t.Fatal("did not expect quiet hours")
	}
}
