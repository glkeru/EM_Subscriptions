package emsub

import (
	"fmt"
	"time"
)

const form = "01-2006"

// парсинг месяца в дату
func ParseDate(s string) (time.Time, error) {
	t, err := time.Parse(form, s)
	if err != nil {
		return t, fmt.Errorf("date parse error: %w", err)
	}
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), nil
}
