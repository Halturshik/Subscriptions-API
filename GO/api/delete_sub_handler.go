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
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// @Summary Удалить подписку
// @Description Удаляет подписку пользователя по дате начала действия
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param user_id path string true "UUID пользователя"
// @Param service_name path string true "Название сервиса"
// @Param body body api.DeleteSubRequest true "Дата начала подписки для удаления"
// @Success 200 {object} api.DeleteSubResponse "Подписка успешно удалена"
// @Failure 400 {object} api.ErrorResponse "Некорректные данные запроса"
// @Failure 404 {object} api.ErrorResponse "Подписка не найдена"
// @Failure 500 {object} api.ErrorResponse "Внутренняя ошибка сервера"
// @Router /users/{user_id}/subscriptions/{service_name} [delete]
func (api *API) DeleteSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "user_id")
	serviceName := strings.TrimSpace(chi.URLParam(r, "service_name"))

	if strings.TrimSpace(userIDStr) == "" || serviceName == "" {
		logger.Warn("Ошибка: не указан uuid пользователя или название сервиса")
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Не указан идентификатор пользователя или название сервиса подписки"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		logger.Warn("Ошибка: некорректный формат uuid: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Некорректный формат идентификатора пользователя"})
		return
	}

	reSN := regexp.MustCompile(`^[A-Za-z0-9 ]+$`)
	if !reSN.MatchString(serviceName) {
		logger.Warn("Ошибка: в названии сервиса используются недопустимые символы")
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Недопустимое название сервиса: используйте только буквы, цифры и пробелы"})
		return
	}

	type deleteReq struct {
		StartDate string `json:"start_date"`
	}

	var req deleteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Ошибка: не удалось прочитать тело запроса: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Некорректно оформлено тело запроса"})
		return
	}

	if strings.TrimSpace(req.StartDate) == "" {
		logger.Warn("Ошибка: не указана дата начала подписки для удаления")
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Не указана дата начала действия подписки, которую вы хотите удалить"})
		return
	}

	startDate, err := time.Parse("01-2006", req.StartDate)
	if err != nil {
		logger.Warn("Ошибка: некорректный формат даты начала подписки для удаления")
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Неверный формат даты начала действия подписки (используйте месяц-год)"})
		return
	}

	err = api.Store.DeleteSubscription(r.Context(), userID, serviceName, startDate)
	if err != nil {
		if errors.Is(err, database.ErrSubNotFound) {
			logger.Warn("Ошибка: подписка не найдена")
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "Подписка не найдена"})
			return
		}
		logger.Error("Ошибка при удалении подписки: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Ошибка при удалении подписки. Повторите попытку позже"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "Подписка успешно удалена"})
	logger.Info("Удалена подписка пользователя %s на сервис %s с датой начала %s", userID, serviceName, req.StartDate)
}
