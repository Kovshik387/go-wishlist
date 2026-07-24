package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/example/wishtrack/internal/bot"
	"github.com/example/wishtrack/internal/store"
)

type Worker struct {
	Store        *store.Store
	Bot          *bot.Client
	Logger       *slog.Logger
	DigestWindow time.Duration
	PollInterval time.Duration
	BotUsername  string
	AppShortName string
	Now          func() time.Time
}

type messagePayload struct {
	AuthorName string   `json:"authorName"`
	WishTitles []string `json:"wishTitles"`
	WishCount  int      `json:"wishCount"`
	PriceMinor *int64   `json:"priceMinor"`
	Currency   string   `json:"currency"`
	DeepLink   string   `json:"deepLink"`
}

func (w *Worker) Run(ctx context.Context) error {
	interval := w.PollInterval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	w.Logger.Info("notification worker started", "poll_interval", interval.String())
	for {
		if err := w.Tick(ctx); err != nil && !errors.Is(err, context.Canceled) {
			w.Logger.Error("notification worker tick failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w *Worker) Tick(ctx context.Context) error {
	_, err := w.Store.PrepareOutbox(ctx, w.DigestWindow, w.BotUsername, w.AppShortName, 50)
	if err != nil {
		return fmt.Errorf("prepare outbox: %w", err)
	}
	for processed := 0; processed < 50; processed++ {
		delivery, err := w.Store.ClaimDelivery(ctx)
		if errors.Is(err, store.ErrNotFound) {
			return nil
		}
		if errors.Is(err, store.ErrConflict) {
			continue
		}
		if err != nil {
			return fmt.Errorf("claim delivery: %w", err)
		}
		now := time.Now().UTC()
		if w.Now != nil {
			now = w.Now().UTC()
		}
		if until, quiet := QuietUntil(now, delivery.Timezone, delivery.Prefs); quiet {
			if err := w.Store.DeferDelivery(ctx, delivery.ID, until); err != nil {
				return err
			}
			continue
		}
		var payload messagePayload
		if err := json.Unmarshal(delivery.Payload, &payload); err != nil {
			_ = w.Store.RetryDelivery(ctx, delivery.ID, 10, "invalid delivery payload", 0)
			continue
		}
		request := bot.SendMessageRequest{
			ChatID: delivery.TelegramID,
			Text: renderMessage(payload),
			ParseMode: "HTML",
			ReplyMarkup: bot.InlineKeyboardMarkup{InlineKeyboard: [][]bot.InlineKeyboardButton{{
				{Text: "Открыть желание", WebApp: &bot.WebAppInfo{URL: payload.DeepLink}},
			}}},
		}
		err = w.Bot.SendMessage(ctx, request)
		switch {
		case err == nil:
			err = w.Store.MarkDeliverySent(ctx, delivery.ID)
		case bot.IsBlocked(err):
			err = w.Store.MarkDeliveryDisabled(ctx, delivery.ID, delivery.UserID, err.Error())
		default:
			err = w.Store.RetryDelivery(ctx, delivery.ID, delivery.Attempts, err.Error(), bot.RetryAfter(err))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func renderMessage(payload messagePayload) string {
	author := html.EscapeString(payload.AuthorName)
	if payload.WishCount > 1 {
		return fmt.Sprintf("✨ <b>%s</b> добавил%s %d новых желания",
			author, genderEnding(author), payload.WishCount)
	}
	title := "новое желание"
	if len(payload.WishTitles) > 0 {
		title = html.EscapeString(payload.WishTitles[0])
	}
	message := fmt.Sprintf("✨ <b>%s</b> добавил%s желание\n\n<b>%s</b>",
		author, genderEnding(author), title)
	if payload.PriceMinor != nil {
		message += "\n" + formatPrice(*payload.PriceMinor, payload.Currency)
	}
	return message
}

func genderEnding(_ string) string {
	return ""
}

func formatPrice(minor int64, currency string) string {
	symbol := map[string]string{"RUB": "₽", "USD": "$", "EUR": "€"}[currency]
	if symbol == "" {
		symbol = currency
	}
	return strconv.FormatInt(minor/100, 10) + " " + symbol
}

func QuietUntil(now time.Time, timezone string, prefs store.NotificationPreferences) (time.Time, bool) {
	if !prefs.QuietHoursEnabled {
		return time.Time{}, false
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		location = time.UTC
	}
	local := now.In(location)
	startHour, startMinute := parseClock(prefs.QuietStart)
	endHour, endMinute := parseClock(prefs.QuietEnd)
	start := time.Date(local.Year(), local.Month(), local.Day(), startHour, startMinute, 0, 0, location)
	end := time.Date(local.Year(), local.Month(), local.Day(), endHour, endMinute, 0, 0, location)
	if !end.After(start) {
		if local.Before(end) {
			start = start.AddDate(0, 0, -1)
		} else {
			end = end.AddDate(0, 0, 1)
		}
	}
	if !local.Before(start) && local.Before(end) {
		return end.UTC(), true
	}
	return time.Time{}, false
}

func parseClock(value string) (int, int) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, 0
	}
	hour, _ := strconv.Atoi(parts[0])
	minute, _ := strconv.Atoi(parts[1])
	return hour, minute
}
