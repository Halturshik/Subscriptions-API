package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (s *Store) UpdateSubscription(ctx context.Context, userID uuid.UUID, serviceName string, newPrice *int, newEndDate *time.Time, newEndDateProvided bool) (priceChanged bool, endDateChanged bool, opType string, err error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return false, false, "", err
	}
	defer tx.Rollback()

	var current Subs
	query := `
		SELECT id, user_id, service_name, price, start_date, end_date
		FROM subscriptions
		WHERE user_id = $1
		  AND service_name = $2
		  AND end_date >= CURRENT_DATE
	`
	err = tx.QueryRowContext(ctx, query, userID, serviceName).Scan(
		&current.ID, &current.UserID, &current.ServiceName,
		&current.Price, &current.StartDate, &current.EndDate,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return false, false, "", ErrSubNotFound
	}
	if err != nil {
		return false, false, "", err
	}

	today := time.Now()
	currentMonthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
	endOfPrevMonth := time.Date(today.Year(), today.Month(), 0, 23, 59, 59, 0, today.Location())
	firstNextMonth := time.Date(today.Year(), today.Month()+1, 1, 0, 0, 0, 0, today.Location())
	endOfCurrentMonth := time.Date(today.Year(), today.Month()+1, 0, 23, 59, 59, 0, today.Location())

	var lastPriceID int
	var lastPrice int
	var lastValidTo *time.Time
	lastPriceQuery := `
		SELECT id, price, valid_to
		FROM subscription_prices
		WHERE subscription_id = $1
		ORDER BY valid_from DESC
		LIMIT 1
	`
	err = tx.QueryRowContext(ctx, lastPriceQuery, current.ID).Scan(&lastPriceID, &lastPrice, &lastValidTo)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, false, "", err
	}

	priceChanged = false
	endDateChanged = false

	if newPrice == nil && newEndDateProvided {
		if !current.EndDate.Equal(*newEndDate) {
			updateSub := `UPDATE subscriptions SET end_date=$1 WHERE id=$2`
			if _, err := tx.ExecContext(ctx, updateSub, newEndDate, current.ID); err != nil {
				return false, false, "", err
			}

			var lastPriceID int
			lastQuery := `
			SELECT id
			FROM subscription_prices
			WHERE subscription_id=$1
			ORDER BY valid_from DESC
			LIMIT 1
		`
			if err := tx.QueryRowContext(ctx, lastQuery, current.ID).Scan(&lastPriceID); err == nil {
				updateValidTo := `UPDATE subscription_prices SET valid_to=$1 WHERE id=$2`
				if _, err := tx.ExecContext(ctx, updateValidTo, newEndDate, lastPriceID); err != nil {
					return false, false, "", err
				}
			}

			return false, true, "date_change", tx.Commit()
		}

		return false, false, "", tx.Commit()
	}

	if newPrice != nil && *newPrice > current.Price {
		priceChanged = true

		delFuture := `DELETE FROM subscription_prices WHERE subscription_id = $1 AND valid_from > CURRENT_DATE`
		if _, err := tx.ExecContext(ctx, delFuture, current.ID); err != nil {
			return false, false, "", err
		}

		var lastPriceID int
		var lastPrice int
		var lastPreviousPrice *int
		var lastValidFrom time.Time
		lastPriceQuery := `
		SELECT id, price, previous_price, valid_from
		FROM subscription_prices
		WHERE subscription_id = $1
		ORDER BY valid_from DESC
		LIMIT 1
	`
		err := tx.QueryRowContext(ctx, lastPriceQuery, current.ID).Scan(
			&lastPriceID, &lastPrice, &lastPreviousPrice, &lastValidFrom,
		)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return false, false, "", err
		}

		if !lastValidFrom.Before(currentMonthStart) {
			updateSubQuery := `UPDATE subscriptions SET price=$1`
			args := []any{*newPrice}

			if newEndDateProvided && !current.EndDate.Equal(*newEndDate) {
				updateSubQuery += ", end_date=$2"
				args = append(args, newEndDate)
				endDateChanged = true
			}

			updateSubQuery += fmt.Sprintf(" WHERE id=$%d", len(args)+1)
			args = append(args, current.ID)

			if _, err := tx.ExecContext(ctx, updateSubQuery, args...); err != nil {
				return false, false, "", err
			}

			effectiveValidTo := current.EndDate
			if newEndDateProvided && !current.EndDate.Equal(*newEndDate) {
				effectiveValidTo = newEndDate
			}

			if lastPreviousPrice == nil {
				updatePriceQuery := `UPDATE subscription_prices SET price=$1, valid_to=$2 WHERE id=$3`
				argsPrice := []any{*newPrice, effectiveValidTo, lastPriceID}

				if _, err := tx.ExecContext(ctx, updatePriceQuery, argsPrice...); err != nil {
					return false, false, "", err
				}

			} else {
				updatePriceQuery := `UPDATE subscription_prices SET price=$1, valid_from=$2, valid_to=$3 WHERE id=$4`
				argsPrice := []any{*newPrice, today, effectiveValidTo, lastPriceID}

				if _, err := tx.ExecContext(ctx, updatePriceQuery, argsPrice...); err != nil {
					return false, false, "", err
				}
			}

			return priceChanged, endDateChanged, "upgrade", tx.Commit()
		}

		updatePrevValidTo := `
		UPDATE subscription_prices
		SET valid_to = $1
		WHERE id = $2 AND valid_from < $3
	`
		if _, err := tx.ExecContext(ctx, updatePrevValidTo, endOfPrevMonth, lastPriceID, currentMonthStart); err != nil {
			return false, false, "", err
		}

		updateSubQuery := `UPDATE subscriptions SET price=$1`
		args := []any{*newPrice}
		if newEndDateProvided && !current.EndDate.Equal(*newEndDate) {
			updateSubQuery += ", end_date=$2"
			args = append(args, newEndDate)
			endDateChanged = true
		}
		updateSubQuery += fmt.Sprintf(" WHERE id=$%d", len(args)+1)
		args = append(args, current.ID)

		if _, err := tx.ExecContext(ctx, updateSubQuery, args...); err != nil {
			return false, false, "", err
		}

		var validTo *time.Time
		if newEndDateProvided && newEndDate != nil {
			validTo = newEndDate
		} else {
			validTo = current.EndDate
		}

		insertQuery := `
		INSERT INTO subscription_prices(subscription_id, price, previous_price, valid_from, valid_to)
		VALUES($1, $2, $3, $4, $5)
	`
		if _, err := tx.ExecContext(ctx, insertQuery, current.ID, *newPrice, current.Price, today, validTo); err != nil {
			return false, false, "", err
		}

		return priceChanged, endDateChanged, "upgrade", tx.Commit()
	}

	if newPrice != nil && *newPrice < current.Price {
		priceChanged = true
		validFrom := firstNextMonth

		if newEndDateProvided && !current.EndDate.Equal(*newEndDate) {
			updateSubQuery := `UPDATE subscriptions SET end_date=$1 WHERE id=$2`
			if _, err := tx.ExecContext(ctx, updateSubQuery, newEndDate, current.ID); err != nil {
				return false, false, "", err
			}
			endDateChanged = true
		}

		effectiveEndDate := lastValidTo
		if endDateChanged {
			effectiveEndDate = newEndDate
		}

		var futureID int
		var futureValidFrom time.Time
		var futureValidTo *time.Time
		checkFuture := `
			SELECT id, valid_from, valid_to
			FROM subscription_prices
			WHERE subscription_id = $1 AND valid_from > CURRENT_DATE
			ORDER BY valid_from ASC
			LIMIT 1
		`
		err = tx.QueryRowContext(ctx, checkFuture, current.ID).Scan(&futureID, &futureValidFrom, &futureValidTo)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return false, false, "", err
		}

		if futureID != 0 {
			updateFuture := `
				UPDATE subscription_prices
				SET price = $1, previous_price = $2, valid_to = $3
				WHERE id = $4
			`
			if _, err := tx.ExecContext(ctx, updateFuture, *newPrice, current.Price, effectiveEndDate, futureID); err != nil {
				return false, false, "", err
			}
			return priceChanged, endDateChanged, "downgrade", tx.Commit()
		}

		if lastPriceID != 0 {
			closeQuery := `UPDATE subscription_prices SET valid_to=$1 WHERE id=$2`
			if _, err := tx.ExecContext(ctx, closeQuery, endOfCurrentMonth, lastPriceID); err != nil {
				return false, false, "", err
			}
		}

		insertQuery := `
			INSERT INTO subscription_prices(subscription_id, price, previous_price, valid_from, valid_to)
			VALUES($1, $2, $3, $4, $5)
		`
		if _, err := tx.ExecContext(ctx, insertQuery, current.ID, *newPrice, current.Price, validFrom, effectiveEndDate); err != nil {
			return false, false, "", err
		}

		return priceChanged, endDateChanged, "downgrade", tx.Commit()
	}

	if newPrice != nil && *newPrice == current.Price {
		if newEndDateProvided && !current.EndDate.Equal(*newEndDate) {
			updateSub := `UPDATE subscriptions SET end_date=$1 WHERE id=$2`
			if _, err := tx.ExecContext(ctx, updateSub, newEndDate, current.ID); err != nil {
				return false, false, "", err
			}
			endDateChanged = true

			var lastPriceID int
			lastQuery := `
            SELECT id
            FROM subscription_prices
            WHERE subscription_id=$1
            ORDER BY valid_from DESC
            LIMIT 1
        `
			if err := tx.QueryRowContext(ctx, lastQuery, current.ID).Scan(&lastPriceID); err == nil {
				updateValidTo := `UPDATE subscription_prices SET valid_to=$1 WHERE id=$2`
				if _, err := tx.ExecContext(ctx, updateValidTo, newEndDate, lastPriceID); err != nil {
					return false, false, "", err
				}
			}
		}

		var futureID int
		var futureStart time.Time
		var futureValidTo *time.Time
		futureQuery := `
        SELECT id, valid_from, valid_to
        FROM subscription_prices
        WHERE subscription_id=$1 AND valid_from > CURRENT_DATE
        ORDER BY valid_from ASC
        LIMIT 1
    `
		err := tx.QueryRowContext(ctx, futureQuery, current.ID).Scan(&futureID, &futureStart, &futureValidTo)
		if errors.Is(err, sql.ErrNoRows) {
			return priceChanged, endDateChanged, "", tx.Commit()
		}
		if err != nil {
			return false, false, "", err
		}

		if futureStart.Before(firstNextMonth) {
			return false, false, "", errors.New("даунгрейд уже вступил в силу, откат невозможен")
		}

		delQuery := `DELETE FROM subscription_prices WHERE id=$1`
		if _, err := tx.ExecContext(ctx, delQuery, futureID); err != nil {
			return false, false, "", err
		}

		var lastPriceID2 int
		var lastValidTo2 *time.Time
		lastQuery := `
        SELECT id, valid_to
        FROM subscription_prices
        WHERE subscription_id=$1
        ORDER BY valid_from DESC
        LIMIT 1
    `
		if err := tx.QueryRowContext(ctx, lastQuery, current.ID).Scan(&lastPriceID2, &lastValidTo2); err != nil {
			return false, false, "", err
		}

		var effectiveEndDate *time.Time
		if endDateChanged {
			effectiveEndDate = newEndDate
		} else {
			effectiveEndDate = current.EndDate
		}

		updateValidTo := `UPDATE subscription_prices SET valid_to=$1 WHERE id=$2`
		if _, err := tx.ExecContext(ctx, updateValidTo, effectiveEndDate, lastPriceID2); err != nil {
			return false, false, "", err
		}

		return priceChanged, endDateChanged, "rollback", tx.Commit()
	}

	return false, false, "", nil
}
