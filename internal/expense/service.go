package expense

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/carvalhocaio/cost-cap-api/internal/validation"
)

type Repository interface {
	Create(ctx context.Context, e Expense) (Expense, error)
	Get(ctx context.Context, userID, id uuid.UUID) (Expense, error)
	Update(ctx context.Context, e Expense) (Expense, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
	List(ctx context.Context, criteria ListCriteria) ([]Expense, error)
}

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository, now func() time.Time) *Service {
	return &Service{repo: repo, now: now}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, draft Draft) (Expense, error) {
	parser := s.newParser()

	created := Expense{
		UserID:      userID,
		Description: parser.description(draft.Description),
		AmountCents: parser.amount(draft.AmountCents),
		Category:    parser.category("category", draft.Category),
		SpentOn:     parser.today,
	}
	if draft.SpentOn != "" {
		created.SpentOn = parser.spentOn(draft.SpentOn)
	}

	if err := parser.errs.OrNil(); err != nil {
		return Expense{}, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return Expense{}, fmt.Errorf("generate expense id: %w", err)
	}
	created.ID = id

	return s.repo.Create(ctx, created)
}

func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (Expense, error) {
	return s.repo.Get(ctx, userID, id)
}

func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, patch Patch) (Expense, error) {
	if patch.isEmpty() {
		return Expense{}, validation.Errors{{Field: "body", Message: "must contain at least one field to update"}}
	}

	updated, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		return Expense{}, err
	}

	parser := s.newParser()
	if patch.Description != nil {
		updated.Description = parser.description(*patch.Description)
	}
	if patch.AmountCents != nil {
		updated.AmountCents = parser.amount(*patch.AmountCents)
	}
	if patch.Category != nil {
		updated.Category = parser.category("category", *patch.Category)
	}
	if patch.SpentOn != nil {
		updated.SpentOn = parser.spentOn(*patch.SpentOn)
	}

	if err := parser.errs.OrNil(); err != nil {
		return Expense{}, err
	}

	return s.repo.Update(ctx, updated)
}

func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.Delete(ctx, userID, id)
}

func (s *Service) List(ctx context.Context, userID uuid.UUID, params ListParams) (Page, error) {
	parser := s.newParser()

	criteria := ListCriteria{
		UserID:   userID,
		Range:    parser.dateRange(params.Period, params.From, params.To),
		Category: parser.optionalCategory(params.Category),
		After:    parser.cursor(params.Cursor),
	}
	limit := parser.limit(params.Limit)

	if err := parser.errs.OrNil(); err != nil {
		return Page{}, err
	}
	criteria.Limit = limit + 1

	expenses, err := s.repo.List(ctx, criteria)
	if err != nil {
		return Page{}, fmt.Errorf("list expenses: %w", err)
	}

	return newPage(expenses, limit), nil
}

func (s *Service) newParser() *fieldParser {
	return &fieldParser{today: dateOf(s.now())}
}
