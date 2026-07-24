package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/example/wishtrack/internal/bot"
	"github.com/example/wishtrack/internal/store"
)

type telegramUpdate struct {
	UpdateID     int64            `json:"update_id"`
	Message      *telegramMessage `json:"message"`
	MyChatMember *struct {
		From          telegramSender `json:"from"`
		NewChatMember struct {
			Status string `json:"status"`
		} `json:"new_chat_member"`
	} `json:"my_chat_member"`
}

type telegramMessage struct {
	Text string `json:"text"`
	Chat struct {
		ID int64 `json:"id"`
	} `json:"chat"`
	From telegramSender `json:"from"`
}

type telegramSender struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	LanguageCode string `json:"language_code"`
}

func (s *Server) telegramWebhook(w http.ResponseWriter, r *http.Request) {
	if s.Config.TelegramWebhookSecret == "" ||
		!secureEqual(r.Header.Get("X-Telegram-Bot-Api-Secret-Token"), s.Config.TelegramWebhookSecret) {
		writeError(w, http.StatusUnauthorized, "INVALID_WEBHOOK_SECRET", "Некорректный секрет webhook")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 512<<10)
	var update telegramUpdate
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&update); err != nil || update.UpdateID == 0 {
		writeError(w, http.StatusBadRequest, "INVALID_UPDATE", "Некорректное обновление Telegram")
		return
	}
	claimed, err := s.Store.ClaimTelegramUpdate(r.Context(), update.UpdateID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if !claimed {
		w.WriteHeader(http.StatusOK)
		return
	}
	if update.MyChatMember != nil {
		status := update.MyChatMember.NewChatMember.Status
		allowed := status == "member" || status == "administrator"
		_ = s.Store.SetBotWriteAllowed(r.Context(), update.MyChatMember.From.ID, allowed)
	}
	if update.Message != nil && strings.HasPrefix(update.Message.Text, "/") {
		s.handleBotCommand(r.Context(), *update.Message)
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleBotCommand(ctx context.Context, message telegramMessage) {
	user, err := s.Store.UpsertTelegramUser(ctx, store.TelegramUser{
		ID: message.From.ID, Username: message.From.Username, FirstName: message.From.FirstName,
		LastName: message.From.LastName, LanguageCode: message.From.LanguageCode,
	})
	if err != nil {
		s.Logger.Error("upsert bot user", "error", err)
		return
	}
	_ = s.Store.SetBotWriteAllowed(ctx, message.From.ID, true)
	fields := strings.Fields(message.Text)
	command := strings.ToLower(fields[0])
	if at := strings.Index(command, "@"); at >= 0 {
		command = command[:at]
	}
	text := "WishTrack помогает собирать желания и делиться ими с друзьями."
	buttonText := "Открыть WishTrack"
	buttonURL := s.Config.PublicURL
	switch command {
	case "/start":
		if len(fields) > 1 && validWishlistStartParam(fields[1]) {
			text = "Вам отправили список желаний. Откройте его кнопкой ниже."
			buttonText = "Открыть список"
			buttonURL = webAppURL(s.Config.PublicURL, fields[1])
		} else {
			text = "Привет, " + user.DisplayName + "! ✨\n\nСоберите желания в одном месте, поделитесь списком и сохраните сюрприз."
		}
	case "/app":
		text = "Откройте WishTrack — ваши списки уже ждут."
	case "/notifications":
		text = "Уведомления включены. Тонкие настройки и тихие часы доступны в профиле приложения."
	case "/help":
		text = "Команды:\n/app — открыть приложение\n/notifications — включить уведомления\n/help — эта подсказка"
	default:
		text = "Не знаю такую команду. Нажмите кнопку ниже или отправьте /help."
	}
	request := bot.SendMessageRequest{
		ChatID: message.Chat.ID,
		Text:   text,
		ReplyMarkup: bot.InlineKeyboardMarkup{InlineKeyboard: [][]bot.InlineKeyboardButton{{
			{Text: buttonText, WebApp: &bot.WebAppInfo{URL: buttonURL}},
		}}},
	}
	sendCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if err := s.Bot.SendMessage(sendCtx, request); err != nil {
		s.Logger.Warn("send bot command response", "error", err, "command", command)
	}
}

func validWishlistStartParam(value string) bool {
	if !strings.HasPrefix(value, "wishlist_") || len(value) > 64 {
		return false
	}
	token := strings.TrimPrefix(value, "wishlist_")
	if token == "" {
		return false
	}
	for _, char := range token {
		if (char < 'a' || char > 'z') &&
			(char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') &&
			char != '_' && char != '-' {
			return false
		}
	}
	return true
}

func webAppURL(publicURL, startParam string) string {
	parsed, err := url.Parse(publicURL)
	if err != nil {
		return publicURL
	}
	query := parsed.Query()
	query.Set("tgWebAppStartParam", startParam)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
