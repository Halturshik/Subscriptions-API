package database

import (
	"context"
	"time"
)

func (s *Store) CreateSubscription(ctx context.Context, sub *Subs) error {
	if sub.EndDate == nil {
		t := time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC)
		sub.EndDate = &t
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	today := time.Now().Truncate(24 * time.Hour)

	if !sub.StartDate.Before(today) {
		var activeCount int
		activeConflictQuery := `
			SELECT COUNT(1)
			FROM subscriptions
			WHERE user_id = $1 AND service_name = $2
			  AND end_date >= CURRENT_DATE
		`
		err = tx.QueryRowContext(ctx, activeConflictQuery, sub.UserID, sub.ServiceName).Scan(&activeCount)
		if err != nil {
			return err
		}
		if activeCount > 0 {
			return ErrSubIsExist
		}
	}

	var conflictCount int
	dateConflictQuery := `
		SELECT COUNT(1)
		FROM subscriptions
		WHERE user_id = $1 AND service_name = $2
		  AND NOT ($3 > end_date OR $4 < start_date)
	`

	err = tx.QueryRowContext(ctx, dateConflictQuery, sub.UserID, sub.ServiceName, sub.StartDate, sub.EndDate).Scan(&conflictCount)
	if err != nil {
		return err
	}
	if conflictCount > 0 {
		return ErrSubOverlapExist
	}

	query := `
		INSERT INTO subscriptions (user_id, service_name, price, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var subID int
	err = tx.QueryRowContext(ctx, query, sub.UserID, sub.ServiceName, sub.Price, sub.StartDate, sub.EndDate).Scan(&subID)
	if err != nil {
		return err
	}

	priceQuery := `
		INSERT INTO subscription_prices (subscription_id, price, valid_from, valid_to)
		VALUES ($1, $2, $3, $4)
	`

	_, err = tx.ExecContext(ctx, priceQuery, subID, sub.Price, sub.StartDate, sub.EndDate)
	if err != nil {
		return err
	}
	return tx.Commit()

}
