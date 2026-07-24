package store

import (
	"context"
	"fmt"
)

func (s *Store) NotificationPreferences(ctx context.Context, userID string) (NotificationPreferences, error) {
	if err := s.ensureNotificationPreferences(ctx, userID); err != nil {
		return NotificationPreferences{}, err
	}
	var prefs NotificationPreferences
	var enabled, newWishes, newLists, reminders, reservations, quiet int
	err := s.DB.QueryRowContext(ctx, `
		SELECT enabled, new_wishes, new_wishlists, event_reminders, reservation_updates,
			quiet_hours_enabled, quiet_start, quiet_end
		FROM notification_preferences WHERE user_id = ?`, userID).Scan(
		&enabled, &newWishes, &newLists, &reminders, &reservations, &quiet,
		&prefs.QuietStart, &prefs.QuietEnd,
	)
	if err != nil {
		return NotificationPreferences{}, isNoRows(err)
	}
	prefs.Enabled = enabled != 0
	prefs.NewWishes = newWishes != 0
	prefs.NewWishlists = newLists != 0
	prefs.EventReminders = reminders != 0
	prefs.ReservationUpdates = reservations != 0
	prefs.QuietHoursEnabled = quiet != 0
	return prefs, nil
}

func (s *Store) UpdateNotificationPreferences(ctx context.Context, userID string, prefs NotificationPreferences) (NotificationPreferences, error) {
	if err := validateClock(prefs.QuietStart); err != nil {
		return NotificationPreferences{}, fmt.Errorf("invalid quietStart")
	}
	if err := validateClock(prefs.QuietEnd); err != nil {
		return NotificationPreferences{}, fmt.Errorf("invalid quietEnd")
	}
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO notification_preferences(
			user_id, enabled, new_wishes, new_wishlists, event_reminders,
			reservation_updates, quiet_hours_enabled, quiet_start, quiet_end, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			enabled = excluded.enabled,
			new_wishes = excluded.new_wishes,
			new_wishlists = excluded.new_wishlists,
			event_reminders = excluded.event_reminders,
			reservation_updates = excluded.reservation_updates,
			quiet_hours_enabled = excluded.quiet_hours_enabled,
			quiet_start = excluded.quiet_start,
			quiet_end = excluded.quiet_end,
			updated_at = excluded.updated_at`,
		userID, boolInt(prefs.Enabled), boolInt(prefs.NewWishes), boolInt(prefs.NewWishlists),
		boolInt(prefs.EventReminders), boolInt(prefs.ReservationUpdates),
		boolInt(prefs.QuietHoursEnabled), prefs.QuietStart, prefs.QuietEnd, nowUTC())
	if err != nil {
		return NotificationPreferences{}, err
	}
	return s.NotificationPreferences(ctx, userID)
}

func (s *Store) ensureNotificationPreferences(ctx context.Context, userID string) error {
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO notification_preferences(user_id, updated_at)
		VALUES (?, ?)
		ON CONFLICT(user_id) DO NOTHING`, userID, nowUTC())
	return err
}

func validateClock(value string) error {
	if len(value) != 5 || value[2] != ':' {
		return fmt.Errorf("invalid time")
	}
	hour := int(value[0]-'0')*10 + int(value[1]-'0')
	minute := int(value[3]-'0')*10 + int(value[4]-'0')
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return fmt.Errorf("invalid time")
	}
	return nil
}
