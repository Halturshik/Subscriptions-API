package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Halturshik/EM-test-task/GO/database"
	"github.com/Halturshik/EM-test-task/GO/logger"
	"github.com/google/uuid"
)

// @Summary Создать подписку
// @Description Создает новую подписку для пользователя
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param subscription body api.CreateSubRequest true "Данные подписки"
// @Success 201 {object} api.CreateSubResponse "Подписка успешно создана"
// @Failure 400 {object} api.ErrorResponse "Ошибка валидации"
// @Failure 409 {object} api.ErrorResponse "Подписка уже существует"
// @Failure 500 {object} api.ErrorResponse "Внутренняя ошибка сервера"
// @Router /subscriptions [post]
func (api *API) CreateSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID      string  `json:"user_id"`
		ServiceName string  `json:"service_name"`
		Price       int     `json:"price"`
		StartDate   string  `json:"start_date"`
		EndDate     *string `json:"end_date"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Ошибка: не удалось прочитать тело запроса: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Некорректно оформлено тело запроса"})
		return
	}

	uid, err := uuid.Parse(req.UserID)
	if err != nil {
		logger.Warn("Ошибка: некорректный формат uuid: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Указан некорректный формат идентификатора пользователя"})
		return
	}

	serviceName := strings.TrimSpace(req.ServiceName)
	if serviceName == "" {
		logger.Warn("Ошибка: не указан сервис подписки")
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Не указан сервис подписки"})
		return
	}

	reSN := regexp.MustCompile(`^[A-Za-z0-9 ]+$`)
	if !reSN.MatchString(serviceName) {
		logger.Warn("Ошибка: в названии сервиса используются недопустимые символы")
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Недопустимое название сервиса: используйте только буквы, цифры и пробелы"})
		return
	}

	validPrices := map[int]bool{50: true, 100: true, 200: true}
	if !validPrices[req.Price] {
		logger.Warn("Ошибка: выбран несуществующий уровень подписки")
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Выберите допустимый уровень подписки: Базовый (50), Продвинутый (100), Премиум (200)"})
		return
	}

	start, err := time.Parse("01-2006", req.StartDate)
	if err != nil {
		logger.Warn("Ошибка: некорректный формат даты начала подписки")
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Неверный формат даты начала действия подписки (используйте месяц-год)"})
		return
	}

	var end *time.Time
	if req.EndDate != nil && *req.EndDate != "" {
		endParsed, err := time.Parse("01-2006", *req.EndDate)
		if err != nil {
			logger.Warn("Ошибка: некорректный формат даты конца подписки")
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Неверный формат даты окончания действия подписки (используйте месяц-год)"})
			return
		}

		if endParsed.Before(start) {
			logger.Warn("Ошибка: дата окончания подписки раньше даты начала")
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Дата окончания действия подписки не может быть раньше даты ее начала действия"})
			return
		}
		endOfMonth := time.Date(endParsed.Year(), endParsed.Month()+1, 0, 23, 59, 59, 0, endParsed.Location())
		end = &endOfMonth
	}
	sub := &database.Subs{
		UserID:      uid,
		ServiceName: serviceName,
		Price:       req.Price,
		StartDate:   start,
		EndDate:     end,
	}

	err = api.Store.CreateSubscription(r.Context(), sub)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrSubIsExist):
			logger.Warn("Ошибка: подписка уже существует")
			writeJSON(w, http.StatusConflict, map[string]any{"error": "Активная подписка на выбранный сервис уже существует"})
			return

		case errors.Is(err, database.ErrSubOverlapExist):
			logger.Warn("Ошибка: добавляемая подписка пересекается с другой")
			writeJSON(w, http.StatusConflict, map[string]any{"error": "Период действия добавляемой подписки пересекается с существующей подпиской"})
			return

		default:
			logger.Error("Ошибка: не удалось создать подписку %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Не удалось создать подписку. Повторите попытку позже"})
			return
		}
	}

	writeJSON(w, http.StatusCreated, map[string]any{"message": "Подписка успешно создана"})
	logger.Info("Создана подписка для пользователя %s на сервис %s", uid, serviceName)

}
