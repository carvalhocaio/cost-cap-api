package expense

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/carvalhocaio/cost-cap-api/internal/validation"
)

const (
	maxDescriptionLength       = 255
	maxAmountCents       int64 = 100_000_000_000
	defaultLimit               = 20
	maxLimit                   = 100
)

type fieldParser struct {
	today time.Time
	errs  validation.Errors
}

func (p *fieldParser) description(raw string) string {
	value := strings.TrimSpace(raw)
	if length := utf8.RuneCountInString(value); length == 0 || length > maxDescriptionLength {
		p.errs.Add("description", fmt.Sprintf("must be between 1 and %d characters", maxDescriptionLength))
	}

	return value
}

func (p *fieldParser) amount(cents int64) int64 {
	if cents <= 0 || cents > maxAmountCents {
		p.errs.Add("amount_cents", fmt.Sprintf("must be between 1 and %d", maxAmountCents))
	}

	return cents
}

func (p *fieldParser) category(field, raw string) Category {
	category := Category(raw)
	if !slices.Contains(categories, category) {
		p.errs.Add(field, "must be one of "+categoryNames())
	}

	return category
}

func (p *fieldParser) optionalCategory(raw string) *Category {
	if raw == "" {
		return nil
	}

	category := p.category("category", raw)
	return &category
}

func (p *fieldParser) date(field, raw string) time.Time {
	value, err := time.Parse(time.DateOnly, raw)
	if err != nil {
		p.errs.Add(field, "must be a date in YYYY-MM-DD format")
	}

	return value
}

func (p *fieldParser) spentOn(raw string) time.Time {
	value := p.date("spent_on", raw)
	if value.After(p.today) {
		p.errs.Add("spent_on", "must not be in the future")
	}

	return value
}

func (p *fieldParser) limit(raw string) int {
	if raw == "" {
		return defaultLimit
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > maxLimit {
		p.errs.Add("limit", fmt.Sprintf("must be an integer between 1 and %d", maxLimit))
	}

	return value
}
