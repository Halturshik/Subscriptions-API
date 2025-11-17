package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func (s *Store) GetSubscriptions(ctx context.Context, userID uuid.UUID, serviceName string, status string, limit int, offset int) ([]Subs, error) {
	query := `
        SELECT id, user_id, service_name, price, start_date, end_date 
        FROM subscriptions 
        WHERE user_id = $1 
    `
	args := []any{userID}

	if strings.TrimSpace(serviceName) != "" {
		query += fmt.Sprintf(" AND service_name = $%d", len(args)+1)
		args = append(args, serviceName)
	}

	if status == "active" {
		query += " AND end_date >= NOW()"
	} else {
		query += " AND end_date < NOW()"
	}

	if status == "active" {
		query += " ORDER BY end_date ASC"
	} else {
		query += " ORDER BY end_date DESC"
	}

	limitPlaceholder := len(args) + 1
	offsetPlaceholder := len(args) + 2

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", limitPlaceholder, offsetPlaceholder)
	args = append(args, limit, offset)

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Subs
	for rows.Next() {
		var s Subs
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.ServiceName, &s.Price, &s.StartDate, &s.EndDate,
		); err != nil {
			return nil, err
		}
		result = append(result, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
