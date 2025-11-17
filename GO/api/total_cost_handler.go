package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Halturshik/EM-test-task/GO/logger"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// @Summary Подсчитать общую стоимость подписки
// @Description Рассчитывает общую стоимость подписки за период
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param user_id path string true "UUID пользователя"
// @Param service_name path string true "Название сервиса"
// @Param body body api.TotalCostRequest true "Период total_from / total_to"
// @Success 200 {object} api.TotalCostResponse "Сообщение с суммой подписки"
// @Failure 400 {object} api.ErrorResponse "Некорректные данные запроса"
// @Failure 500 {object} api.ErrorResponse "Внутренняя ошибка сервера"
// @Router /users/{user_id}/subscriptions/{service_name}/total [post]
func (api *API) GetTotalSubscriptionCostHandler(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		TotalFrom *string `json:"total_from"`
		TotalTo   *string `json:"total_to"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Ошибка: не удалось прочитать тело запроса: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Некорректно оформлено тело запроса"})
		return
	}

	if req.TotalFrom == nil || req.TotalTo == nil || strings.TrimSpace(*req.TotalFrom) == "" || strings.TrimSpace(*req.TotalTo) == "" {
		logger.Warn("Ошибка: не заполнены даты для подсчета стоимости подписки")
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Не указан период для подсчета стоимости подписки"})
		return
	}

	fromDate, err := time.Parse("01-2006", *req.TotalFrom)
	if err != nil {
		logger.Warn("Ошибка: некорректный формат даты начала для подсчета стоимости подписки")
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Неверный формат даты начала периода для подсчета стоимости подписки (используйте месяц-год)"})
		return
	}

	toDateParsed, err := time.Parse("01-2006", *req.TotalTo)
	if err != nil {
		logger.Warn("Ошибка: некорректный формат даты окончания для подсчета стоимости подписки")
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Неверный формат даты окончания периода для подсчета стоимости подписки (используйте месяц-год)"})
		return
	}
	toDate := time.Date(toDateParsed.Year(), toDateParsed.Month()+1, 0, 0, 0, 0, 0, time.UTC)

	if toDate.Before(fromDate) {
		logger.Warn("Ошибка: дата окончания периода раньше даты начала периода для подсчета стоимости подписки")
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Дата окончания периода не может быть раньше даты начала периода для подсчета стоимости подписки"})
		return
	}

	now := time.Now()
	endOfCurrentMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	if toDate.After(endOfCurrentMonth) {
		logger.Warn("Ошибка: дата окончания периода для подсчета стоимости подписки больше текущего месяца")
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Дата окончания периода для подсчета стоимости подписки не может быть больше текущего месяца"})
		return
	}

	totalCost, status, err := api.Store.CalculateTotalSubscriptionCost(r.Context(), userID, serviceName, fromDate, toDate)
	if err != nil {
		logger.Error("Ошибка при расчете стоимости подписок: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Ошибка при расчете стоимости подписок. Повторите попытку позже"})
		return
	}

	var msg string

	switch status {
	case "no_subscription":
		msg = "Подписок не найдено"
	case "no_overlap":
		msg = fmt.Sprintf("Подписка %s не действовала в выбранный период", serviceName)
	case "ok":
		msg = fmt.Sprintf("Общая стоимость подписки %s за указанный период составила: %d", serviceName, totalCost)
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": msg})
	logger.Info("Расчет стоимости подписки для пользователя %s на сервис %s за период %s - %s завершен. Сумма: %d", userID, serviceName, *req.TotalFrom, *req.TotalTo, totalCost)

}
