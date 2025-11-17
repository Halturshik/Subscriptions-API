package api

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Halturshik/EM-test-task/GO/logger"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// @Summary Получить подписки пользователя
// @Description Возвращает список подписок для указанного user_id. Можно фильтровать по статусу и пагинировать.
// @Tags subscriptions
// @Produce json
// @Param user_id path string true "UUID пользователя"
// @Param status query string false "Статус подписки" Enums(active, archived) default(active)
// @Param page query int false "Номер страницы для пагинации" default(1)
// @Success 200 {array} api.SubResponse "Список подписок"
// @Failure 400 {object} api.ErrorResponse "Некорректный UUID пользователя, service_name, статус или номер страницы"
// @Failure 500 {object} api.ErrorResponse "Внутренняя ошибка сервера"
// @Router /users/{user_id}/subscriptions [get]
// @Router /users/{user_id}/subscriptions/{service_name} [get]
func (api *API) GetSubscriptionsHandler(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "user_id")
	serviceName := strings.TrimSpace(chi.URLParam(r, "service_name"))

	if strings.TrimSpace(userIDStr) == "" {
		logger.Warn("Ошибка: не указан uuid пользователя")
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Не указан идентификатор пользователя"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		logger.Warn("Ошибка: некорректный формат uuid: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Некорректный формат идентификатора"})
		return
	}

	status := r.URL.Query().Get("status")
	if status == "" {
		status = "active"
	}
	if status != "active" && status != "archived" {
		logger.Warn("Ошибка: некорректный статус подписки")
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Некорректный статус подписки"})
		return
	}

	if serviceName != "" {
		reSN := regexp.MustCompile(`^[A-Za-z0-9 ]+$`)
		if !reSN.MatchString(serviceName) {
			logger.Warn("Ошибка: в названии сервиса используются недопустимые символы")
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Недопустимое название сервиса: используйте только буквы, цифры и пробелы"})
			return
		}
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	limit := 5
	offset := (page - 1) * limit

	subsFromDB, err := api.Store.GetSubscriptions(r.Context(), userID, serviceName, status, limit, offset)
	if err != nil {
		logger.Error("Ошибка: не удалось вытащить подписку: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Не удалось произвести поиск подписки. Повторите попытку позже"})
		return
	}

	if len(subsFromDB) == 0 {
		logger.Info("Подписок не найдено")
		writeJSON(w, http.StatusOK, map[string]any{"message": "Подписок не найдено"})
		return
	}

	type subsResponse struct {
		ServiceName string  `json:"service_name"`
		Price       int     `json:"price"`
		StartDate   string  `json:"start_date"`
		EndDate     *string `json:"end_date,omitempty"`
	}

	resp := make([]subsResponse, 0, len(subsFromDB))
	for _, s := range subsFromDB {
		var endStr *string
		infiniteDate := time.Date(2099, 12, 31, 0, 0, 0, 0, s.EndDate.Location())
		if !s.EndDate.Equal(infiniteDate) {
			tmp := s.EndDate.Format("01-2006")
			endStr = &tmp
		}
		resp = append(resp, subsResponse{
			ServiceName: s.ServiceName,
			Price:       s.Price,
			StartDate:   s.StartDate.Format("01-2006"),
			EndDate:     endStr,
		})
	}

	writeJSON(w, http.StatusOK, resp)
	logger.Info("Выдано подписок: user=%s service=%s status=%s page=%d count=%d", userID, serviceName, status, page, len(subsFromDB))
}
