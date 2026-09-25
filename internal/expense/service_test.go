package expense_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/carvalhocaio/cost-cap-api/internal/expense"
	"github.com/carvalhocaio/cost-cap-api/internal/validation"
)

var (
	owner         = uuid.Must(uuid.NewV7())
	clancyRelease = date(2024, time.May, 24)
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func ptr[T any](value T) *T {
	return &value
}

type fakeRepository struct {
	stored   map[uuid.UUID]expense.Expense
	listed   []expense.Expense
	criteria expense.ListCriteria
}

func (r *fakeRepository) Create(_ context.Context, e expense.Expense) (expense.Expense, error) {
	r.stored[e.ID] = e
	return e, nil
}

func (r *fakeRepository) Get(_ context.Context, userID, id uuid.UUID) (expense.Expense, error) {
	found, ok := r.stored[id]
	if !ok || found.UserID != userID {
		return expense.Expense{}, expense.ErrNotFound
	}

	return found, nil
}

func (r *fakeRepository) Update(_ context.Context, e expense.Expense) (expense.Expense, error) {
	r.stored[e.ID] = e
	return e, nil
}

func (r *fakeRepository) Delete(_ context.Context, userID, id uuid.UUID) error {
	if _, err := r.Get(context.Background(), userID, id); err != nil {
		return err
	}
	delete(r.stored, id)

	return nil
}

func (r *fakeRepository) List(_ context.Context, criteria expense.ListCriteria) ([]expense.Expense, error) {
	r.criteria = criteria
	return r.listed, nil
}

func newService(today time.Time) (*expense.Service, *fakeRepository) {
	repo := &fakeRepository{stored: map[uuid.UUID]expense.Expense{}}
	clock := func() time.Time { return today.Add(15 * time.Hour) }

	return expense.NewService(repo, clock), repo
}

func assertFieldError(t *testing.T, err error, field string) {
	t.Helper()

	var validationErrs validation.Errors
	if !errors.As(err, &validationErrs) {
		t.Fatalf("error = %v, want validation.Errors", err)
	}

	if !slices.ContainsFunc(validationErrs, func(e validation.FieldError) bool { return e.Field == field }) {
		t.Errorf("errors = %v, want one for field %q", validationErrs, field)
	}
}

func TestCreateDefaultsAndNormalizes(t *testing.T) {
	service, _ := newService(clancyRelease)

	created, err := service.Create(t.Context(), owner, expense.Draft{
		Description: "  Vinyl pressing ",
		AmountCents: 4599,
		Category:    "leisure",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if created.Description != "Vinyl pressing" {
		t.Errorf("Description = %q, want it trimmed", created.Description)
	}
	if !created.SpentOn.Equal(clancyRelease) {
		t.Errorf("SpentOn = %v, want today %v", created.SpentOn, clancyRelease)
	}
	if created.ID == uuid.Nil || created.UserID != owner {
		t.Errorf("identity = %v/%v, want generated id owned by %v", created.ID, created.UserID, owner)
	}
}

func TestCreateRejectsInvalidDrafts(t *testing.T) {
	tests := []struct {
		name      string
		draft     expense.Draft
		wantField string
	}{
		{"blank description", expense.Draft{Description: "   ", AmountCents: 100, Category: "leisure"}, "description"},
		{"zero amount", expense.Draft{Description: "Tickets", AmountCents: 0, Category: "leisure"}, "amount_cents"},
		{"unknown category", expense.Draft{Description: "Tickets", AmountCents: 100, Category: "merch"}, "category"},
		{"future date", expense.Draft{Description: "Tickets", AmountCents: 100, Category: "leisure", SpentOn: "2024-05-25"}, "spent_on"},
		{"malformed date", expense.Draft{Description: "Tickets", AmountCents: 100, Category: "leisure", SpentOn: "24/05/2024"}, "spent_on"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, _ := newService(clancyRelease)

			_, err := service.Create(t.Context(), owner, tt.draft)

			assertFieldError(t, err, tt.wantField)
		})
	}
}

func TestUpdateAppliesOnlyProvidedFields(t *testing.T) {
	service, _ := newService(clancyRelease)
	created, err := service.Create(t.Context(), owner, expense.Draft{Description: "Tour tickets", AmountCents: 12000, Category: "leisure"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := service.Update(t.Context(), owner, created.ID, expense.Patch{AmountCents: ptr(int64(15000))})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	if updated.AmountCents != 15000 {
		t.Errorf("AmountCents = %d, want 15000", updated.AmountCents)
	}
	if updated.Description != created.Description || updated.Category != created.Category {
		t.Errorf("untouched fields changed: %+v", updated)
	}
}

func TestUpdateRejections(t *testing.T) {
	service, _ := newService(clancyRelease)
	created, err := service.Create(t.Context(), owner, expense.Draft{Description: "Tour tickets", AmountCents: 12000, Category: "leisure"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	t.Run("empty patch", func(t *testing.T) {
		_, err := service.Update(t.Context(), owner, created.ID, expense.Patch{})
		assertFieldError(t, err, "body")
	})

	t.Run("invalid field", func(t *testing.T) {
		_, err := service.Update(t.Context(), owner, created.ID, expense.Patch{Category: ptr("merch")})
		assertFieldError(t, err, "category")
	})

	t.Run("foreign owner", func(t *testing.T) {
		stranger := uuid.Must(uuid.NewV7())
		_, err := service.Update(t.Context(), stranger, created.ID, expense.Patch{AmountCents: ptr(int64(1))})
		if !errors.Is(err, expense.ErrNotFound) {
			t.Errorf("error = %v, want %v", err, expense.ErrNotFound)
		}
	})
}

func TestListResolvesPeriods(t *testing.T) {
	tests := []struct {
		name   string
		today  time.Time
		params expense.ListParams
		want   *expense.DateRange
	}{
		{"all time", clancyRelease, expense.ListParams{}, nil},
		{"past week", clancyRelease, expense.ListParams{Period: "past_week"}, &expense.DateRange{From: date(2024, time.May, 18), To: clancyRelease}},
		{"past month", clancyRelease, expense.ListParams{Period: "past_month"}, &expense.DateRange{From: date(2024, time.April, 25), To: clancyRelease}},
		{"last three months", clancyRelease, expense.ListParams{Period: "last_3_months"}, &expense.DateRange{From: date(2024, time.February, 25), To: clancyRelease}},
		{"past month clamps short months", date(2025, time.March, 31), expense.ListParams{Period: "past_month"}, &expense.DateRange{From: date(2025, time.March, 1), To: date(2025, time.March, 31)}},
		{"three months crosses year boundary", date(2025, time.January, 15), expense.ListParams{Period: "last_3_months"}, &expense.DateRange{From: date(2024, time.October, 16), To: date(2025, time.January, 15)}},
		{"custom", clancyRelease, expense.ListParams{Period: "custom", From: "2024-01-01", To: "2024-03-31"}, &expense.DateRange{From: date(2024, time.January, 1), To: date(2024, time.March, 31)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, repo := newService(tt.today)

			if _, err := service.List(t.Context(), owner, tt.params); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := repo.criteria.Range
			if (got == nil) != (tt.want == nil) {
				t.Fatalf("Range = %+v, want %+v", got, tt.want)
			}
			if got != nil && (!got.From.Equal(tt.want.From) || !got.To.Equal(tt.want.To)) {
				t.Errorf("Range = %v..%v, want %v..%v", got.From, got.To, tt.want.From, tt.want.To)
			}
		})
	}
}

func TestListRejectsInvalidParams(t *testing.T) {
	tests := []struct {
		name      string
		params    expense.ListParams
		wantField string
	}{
		{"unknown period", expense.ListParams{Period: "fortnight"}, "period"},
		{"dates without custom", expense.ListParams{From: "2024-01-01"}, "period"},
		{"custom without to", expense.ListParams{Period: "custom", From: "2024-01-01"}, "period"},
		{"inverted range", expense.ListParams{Period: "custom", From: "2024-03-01", To: "2024-01-01"}, "to"},
		{"malformed from", expense.ListParams{Period: "custom", From: "yesterday", To: "2024-01-01"}, "from"},
		{"unknown category", expense.ListParams{Category: "merch"}, "category"},
		{"limit out of bounds", expense.ListParams{Limit: "500"}, "limit"},
		{"tampered cursor", expense.ListParams{Cursor: "not-a-cursor"}, "cursor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, _ := newService(clancyRelease)

			_, err := service.List(t.Context(), owner, tt.params)

			assertFieldError(t, err, tt.wantField)
		})
	}
}

func TestListPaginatesWithCursor(t *testing.T) {
	service, repo := newService(clancyRelease)
	newest := expense.Expense{ID: uuid.Must(uuid.NewV7()), SpentOn: date(2024, time.May, 24)}
	middle := expense.Expense{ID: uuid.Must(uuid.NewV7()), SpentOn: date(2024, time.May, 20)}
	oldest := expense.Expense{ID: uuid.Must(uuid.NewV7()), SpentOn: date(2024, time.May, 10)}
	repo.listed = []expense.Expense{newest, middle, oldest}

	first, err := service.List(t.Context(), owner, expense.ListParams{Limit: "2"})
	if err != nil {
		t.Fatalf("first page: %v", err)
	}

	if repo.criteria.Limit != 3 {
		t.Errorf("repository limit = %d, want limit+1 = 3", repo.criteria.Limit)
	}
	if len(first.Expenses) != 2 || first.Next == nil || first.Next.ID != middle.ID {
		t.Fatalf("first page = %d expenses, next %+v, want 2 and a cursor at the middle expense", len(first.Expenses), first.Next)
	}

	repo.listed = []expense.Expense{oldest}
	second, err := service.List(t.Context(), owner, expense.ListParams{Limit: "2", Cursor: first.Next.String()})
	if err != nil {
		t.Fatalf("second page: %v", err)
	}

	after := repo.criteria.After
	if after == nil || after.ID != middle.ID || !after.SpentOn.Equal(middle.SpentOn) {
		t.Errorf("cursor did not round-trip: got %+v, want %+v", after, first.Next)
	}
	if second.Next != nil {
		t.Errorf("last page has a next cursor: %+v", second.Next)
	}
}
