package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/example/wishtrack/internal/store"
)

var errUnauthorized = errors.New("unauthorized")

type apiError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	var response apiError
	response.Error.Code = code
	response.Error.Message = message
	writeJSON(w, status, response)
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Объект не найден")
	case errors.Is(err, store.ErrForbidden):
		writeError(w, http.StatusForbidden, "FORBIDDEN", "У вас нет доступа")
	case errors.Is(err, store.ErrVersionConflict):
		writeError(w, http.StatusConflict, "VERSION_CONFLICT", "Данные уже изменились. Обновите страницу")
	case errors.Is(err, store.ErrAlreadyReserved):
		writeError(w, http.StatusConflict, "ALREADY_RESERVED", "Это желание уже забронировали")
	case errors.Is(err, store.ErrOwnWish):
		writeError(w, http.StatusUnprocessableEntity, "OWN_WISH", "Нельзя бронировать своё желание")
	case errors.Is(err, store.ErrReservationsOff):
		writeError(w, http.StatusUnprocessableEntity, "RESERVATIONS_DISABLED", "Автор отключил бронирование")
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL", "Что-то пошло не так")
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "INVALID_JSON", "Тело запроса пустое")
		} else {
			writeError(w, http.StatusBadRequest, "INVALID_JSON", "Некорректные данные запроса")
		}
		return false
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "В запросе должен быть один JSON-объект")
		return false
	}
	return true
}
