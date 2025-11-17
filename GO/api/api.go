package api

// @title Subscriptions API
// @version 2.0
// @description REST API для управления онлайн-подписками пользователей
// @host localhost:8080
// @BasePath /

// @contact.name Artem
// @contact.email disaer21@yandex.ru

import (
	"context"
	"time"

	"github.com/Halturshik/EM-test-task/GO/database"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	httpSwagger "github.com/swaggo/http-swagger"
)

type Store interface {
	CreateSubscription(ctx context.Context, s *database.Subs) error
	UpdateSubscription(ctx context.Context, userID uuid.UUID, serviceName string, newPrice *int, newEndDate *time.Time, newEndDateProvided bool) (bool, bool, string, error)
	DeleteSubscription(ctx context.Context, userID uuid.UUID, serviceName string, startDate time.Time) error
	GetSubscriptions(ctx context.Context, userID uuid.UUID, serviceName string, status string, limit int, offset int) ([]database.Subs, error)
	CalculateTotalSubscriptionCost(ctx context.Context, userID uuid.UUID, serviceName string, from, to time.Time) (int, string, error)
	SyncSubscriptionPrices(ctx context.Context) error
}

type API struct {
	Store Store
}

func NewAPI(store Store) *API {
	return &API{Store: store}
}

func (api *API) Init(r *chi.Mux) {
	r.Route("/subscriptions", func(r chi.Router) {
		r.Post("/", api.CreateSubscriptionHandler)
	})

	r.Route("/users/{user_id}/subscriptions", func(r chi.Router) {
		r.Get("/", api.GetSubscriptionsHandler)
		r.Get("/{service_name}", api.GetSubscriptionsHandler)
		r.Put("/{service_name}", api.UpdateSubscriptionHandler)
		r.Delete("/{service_name}", api.DeleteSubscriptionHandler)
		r.Post("/{service_name}/total", api.GetTotalSubscriptionCostHandler)

	})

	r.Get("/swagger/*", httpSwagger.Handler())

}
