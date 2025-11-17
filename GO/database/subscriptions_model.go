package database

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Subs struct {
	ID          int        `json:"-"`
	UserID      uuid.UUID  `json:"-"`
	ServiceName string     `json:"service_name"`
	Price       int        `json:"price"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     *time.Time `json:"end_date"`
}

type SubsPriceHistory struct {
	ID             int        `json:"-"`
	SubscriptionID int        `json:"-"`
	Price          int        `json:"price"`
	ValidFrom      time.Time  `json:"valid_from"`
	ValidTo        *time.Time `json:"valid_to"`
}

var ErrSubIsExist = errors.New("подписка существует")
var ErrSubOverlapExist = errors.New("подписка пересекается с другой")
var ErrSubNotFound = errors.New("подписка не найдена")
