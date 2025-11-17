package database

import (
	"context"
	"time"

	"github.com/google/uuid"
)

func (s *Store) CalculateTotalSubscriptionCost(ctx context.Context, userID uuid.UUID, serviceName string, from, to time.Time) (int, string, error) {
	var exists bool
	checkQuery := `
		SELECT EXISTS (
			SELECT 1 FROM subscriptions
			WHERE user_id = $1 AND service_name = $2)
	`

	if err := s.DB.QueryRowContext(ctx, checkQuery, userID, serviceName).Scan(&exists); err != nil {
		return 0, "", err
	}

	if !exists {
		return 0, "no_subscription", nil
	}

	now := time.Now()
	endOfCurrentMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC)

	query := `
		SELECT  
			sp.price,
			GREATEST(sp.valid_from, s.start_date, $3) AS overlap_start,
			LEAST(sp.valid_to, s.end_date, $4, $5) AS overlap_end
		FROM subscriptions s
		JOIN subscription_prices sp 
			ON sp.subscription_id = s.id
		WHERE s.user_id = $1
		  AND s.service_name = $2
		  AND s.start_date <= $4
		  AND s.end_date   >= $3
		  AND sp.valid_from <= $4
		  AND sp.valid_to   >= $3
		ORDER BY overlap_start;
	`

	rows, err := s.DB.QueryContext(ctx, query, userID, serviceName, from, to, endOfCurrentMonth)
	if err != nil {
		return 0, "", err
	}
	defer rows.Close()

	total := 0
	hasOverlap := false

	for rows.Next() {
		var price int
		var overlapStart, overlapEnd time.Time

		if err := rows.Scan(&price, &overlapStart, &overlapEnd); err != nil {
			return 0, "", err
		}

		if overlapEnd.Before(overlapStart) {
			continue
		}

		months := countMonths(overlapStart, overlapEnd)
		if months > 0 {
			total += price * months
			hasOverlap = true
		}
	}

	if err := rows.Err(); err != nil {
		return 0, "", err
	}

	if !hasOverlap {
		return 0, "no_overlap", nil
	}

	return total, "ok", nil
}

func countMonths(start, end time.Time) int {
	yearDiff := end.Year() - start.Year()
	monthDiff := int(end.Month()) - int(start.Month())
	return yearDiff*12 + monthDiff + 1
}
