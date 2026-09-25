package expense

import (
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("expense not found")

type Category string

const (
	CategoryGroceries   Category = "groceries"
	CategoryLeisure     Category = "leisure"
	CategoryElectronics Category = "electronics"
	CategoryUtilities   Category = "utilities"
	CategoryClothing    Category = "clothing"
	CategoryHealth      Category = "health"
	CategoryOthers      Category = "others"
)

var categories = []Category{
	CategoryGroceries,
	CategoryLeisure,
	CategoryElectronics,
	CategoryUtilities,
	CategoryClothing,
	CategoryHealth,
	CategoryOthers,
}

func Categories() []Category {
	return slices.Clone(categories)
}

func categoryNames() string {
	names := make([]string, len(categories))
	for i, category := range categories {
		names[i] = string(category)
	}

	return strings.Join(names, ", ")
}

type Expense struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Description string
	AmountCents int64
	Category    Category
	SpentOn     time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Draft struct {
	Description string
	AmountCents int64
	Category    string
	SpentOn     string
}

type Patch struct {
	Description *string
	AmountCents *int64
	Category    *string
	SpentOn     *string
}

func (p Patch) isEmpty() bool {
	return p.Description == nil && p.AmountCents == nil && p.Category == nil && p.SpentOn == nil
}

func dateOf(t time.Time) time.Time {
	year, month, day := t.UTC().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
