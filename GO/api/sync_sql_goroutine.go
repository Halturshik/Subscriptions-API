package api

import (
	"context"
	"time"

	"github.com/Halturshik/EM-test-task/GO/database"
	"github.com/Halturshik/EM-test-task/GO/logger"
)

var monthlySyncCancel context.CancelFunc

func StartMonthlySync(store *database.Store) {
	ctx, cancel := context.WithCancel(context.Background())
	monthlySyncCancel = cancel

	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Info("Фоновая синхронизация подписок остановлена")
				return
			default:
				now := time.Now()
				next := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
				duration := time.Until(next)

				select {
				case <-time.After(duration):
					if err := store.SyncSubscriptionPrices(context.Background()); err != nil {
						logger.Error("Ошибка синхронизации подписок: %v", err)
					} else {
						logger.Info("Синхронизация подписок выполнена успешно")
					}
				case <-ctx.Done():
					logger.Info("Фоновая синхронизация подписок остановлена")
					return
				}
			}
		}
	}()
}

func StopMonthlySync() {
	if monthlySyncCancel != nil {
		monthlySyncCancel()
	}
}
