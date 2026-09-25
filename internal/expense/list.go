package expense

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/google/uuid"
)

const cursorSeparator = "|"

type ListParams struct {
	Period   string
	From     string
	To       string
	Category string
	Cursor   string
	Limit    string
}

type ListCriteria struct {
	UserID   uuid.UUID
	Range    *DateRange
	Category *Category
	After    *Cursor
	Limit    int
}

type Cursor struct {
	SpentOn time.Time
	ID      uuid.UUID
}

func (c Cursor) String() string {
	raw := c.SpentOn.Format(time.DateOnly) + cursorSeparator + c.ID.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

type Page struct {
	Expenses []Expense
	Next     *Cursor
}

func newPage(expenses []Expense, limit int) Page {
	if len(expenses) <= limit {
		return Page{Expenses: expenses}
	}

	page := expenses[:limit]
	last := page[len(page)-1]

	return Page{Expenses: page, Next: &Cursor{SpentOn: last.SpentOn, ID: last.ID}}
}

func (p *fieldParser) cursor(raw string) *Cursor {
	if raw == "" {
		return nil
	}

	cursor, ok := decodeCursor(raw)
	if !ok {
		p.errs.Add("cursor", "is invalid")
		return nil
	}

	return &cursor
}

func decodeCursor(raw string) (Cursor, bool) {
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return Cursor{}, false
	}

	rawDate, rawID, found := strings.Cut(string(decoded), cursorSeparator)
	if !found {
		return Cursor{}, false
	}

	spentOn, err := time.Parse(time.DateOnly, rawDate)
	if err != nil {
		return Cursor{}, false
	}

	id, err := uuid.Parse(rawID)
	if err != nil {
		return Cursor{}, false
	}

	return Cursor{SpentOn: spentOn, ID: id}, true
}
