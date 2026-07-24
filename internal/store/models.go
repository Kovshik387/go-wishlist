package store

import (
	"encoding/json"
	"time"
)

type User struct {
	ID                  string `json:"id"`
	TelegramID          int64  `json:"telegramId,omitempty"`
	Username            string `json:"username"`
	DisplayName         string `json:"displayName"`
	AvatarURL           string `json:"avatarUrl"`
	LanguageCode        string `json:"languageCode"`
	Timezone            string `json:"timezone"`
	BotWriteAllowed     bool   `json:"botWriteAllowed"`
	OnboardingCompleted bool   `json:"onboardingCompleted"`
	CreatedAt           string `json:"createdAt"`
}

type Wishlist struct {
	ID                    string `json:"id"`
	OwnerID               string `json:"ownerId"`
	Title                 string `json:"title"`
	Description           string `json:"description"`
	Emoji                 string `json:"emoji"`
	CoverURL              string `json:"coverUrl"`
	Occasion              string `json:"occasion"`
	EventDate             string `json:"eventDate,omitempty"`
	Visibility            string `json:"visibility"`
	AllowReservations     bool   `json:"allowReservations"`
	OwnerSeesReservations bool   `json:"ownerSeesReservations"`
	PublicToken           string `json:"publicToken,omitempty"`
	Version               int    `json:"version"`
	WishCount             int    `json:"wishCount"`
	CreatedAt             string `json:"createdAt"`
	UpdatedAt             string `json:"updatedAt"`
	Owner                 *User  `json:"owner,omitempty"`
	Wishes                []Wish `json:"wishes,omitempty"`
}

type Wish struct {
	ID             string            `json:"id"`
	WishlistID     string            `json:"wishlistId"`
	ProductURL     string            `json:"productUrl"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	ImageURL       string            `json:"imageUrl"`
	PriceMinor     *int64             `json:"priceMinor,omitempty"`
	Currency       string            `json:"currency"`
	Priority       string            `json:"priority"`
	Quantity       int               `json:"quantity"`
	Attributes     map[string]string `json:"attributes"`
	StoreDomain    string            `json:"storeDomain"`
	Version        int               `json:"version"`
	IsReserved     bool              `json:"isReserved"`
	ReservedByMe   bool              `json:"reservedByMe"`
	ReservedBy     *User             `json:"reservedBy,omitempty"`
	CreatedAt      string            `json:"createdAt"`
	UpdatedAt      string            `json:"updatedAt"`
	Wishlist       *Wishlist         `json:"wishlist,omitempty"`
	Author         *User             `json:"author,omitempty"`
}

type Reservation struct {
	ID        string `json:"id"`
	WishID    string `json:"wishId"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	Wish      Wish   `json:"wish"`
}

type NotificationPreferences struct {
	Enabled              bool   `json:"enabled"`
	NewWishes            bool   `json:"newWishes"`
	NewWishlists         bool   `json:"newWishlists"`
	EventReminders       bool   `json:"eventReminders"`
	ReservationUpdates   bool   `json:"reservationUpdates"`
	QuietHoursEnabled    bool   `json:"quietHoursEnabled"`
	QuietStart           string `json:"quietStart"`
	QuietEnd             string `json:"quietEnd"`
}

type TelegramUser struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	PhotoURL     string `json:"photo_url"`
	LanguageCode string `json:"language_code"`
}

type WishlistInput struct {
	Title                 string `json:"title"`
	Description           string `json:"description"`
	Emoji                 string `json:"emoji"`
	CoverURL              string `json:"coverUrl"`
	Occasion              string `json:"occasion"`
	EventDate             string `json:"eventDate"`
	Visibility            string `json:"visibility"`
	AllowReservations     *bool  `json:"allowReservations"`
	OwnerSeesReservations *bool  `json:"ownerSeesReservations"`
	Version               int    `json:"version"`
}

type WishInput struct {
	ProductURL  string            `json:"productUrl"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	ImageURL    string            `json:"imageUrl"`
	PriceMinor  *int64             `json:"priceMinor"`
	Currency    string            `json:"currency"`
	Priority    string            `json:"priority"`
	Quantity    int               `json:"quantity"`
	Attributes  map[string]string `json:"attributes"`
	StoreDomain string            `json:"storeDomain"`
	Version     int               `json:"version"`
}

type Delivery struct {
	ID         string
	EventID    string
	UserID     string
	TelegramID int64
	Payload    json.RawMessage
	Attempts   int
	Timezone   string
	Prefs      NotificationPreferences
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
