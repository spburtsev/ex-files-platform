package handlers

import (
	"fmt"
	"math"
	"time"

	"github.com/spburtsev/ex-files-backend/oapi"
)

const (
	defaultPage    = 1
	defaultPerPage = 20
	maxPerPage     = 100
)

// resolvePagination clamps the optional page/per_page parameters to the
// project-wide defaults and returns the (page, perPage, offset) triple.
func resolvePagination(page, perPage oapi.OptInt32) (int, int, int) {
	p := defaultPage
	pp := defaultPerPage
	if page.IsSet() && page.Value > 0 {
		p = int(page.Value)
	}
	if perPage.IsSet() {
		v := int(perPage.Value)
		if v > 0 && v <= maxPerPage {
			pp = v
		}
	}
	return p, pp, (p - 1) * pp
}

// totalPages returns the number of pages for total items at the given size.
// At least one page is reported even when total is zero so callers can render
// a consistent paginator.
func totalPages(total int64, perPage int) int {
	if perPage <= 0 {
		return 0
	}
	if total == 0 {
		return 1
	}
	return int(math.Ceil(float64(total) / float64(perPage)))
}

// optInt64 wraps an int64 in an OptInt64.
func optInt64(v int64) oapi.OptInt64 {
	return oapi.NewOptInt64(v)
}

// optInt32 wraps an int from a clamped int (e.g. page) into OptInt32.
func optInt32(v int) oapi.OptInt32 {
	return oapi.NewOptInt32(int32(v))
}

// parseTime tolerates a few common shapes for incoming date strings.
func parseTime(s string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised time format: %s", s)
}
