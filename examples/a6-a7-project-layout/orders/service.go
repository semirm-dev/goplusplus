package orders

import "context"

// There is no need for a type OrderService interface to wrap this service.
type Service struct {
	repo      Repository
	discounts Discounts
}

// Ports: needed for database operations.
// We can pass a mock in tests.
type Repository interface {
	Save(ctx context.Context, o Order) error
	Find(ctx context.Context, id string) (Order, error)
}

// Ports: needed for discount operations.
// We can pass a mock in tests.
type Discounts interface {
	Percent(code string) int64
}

func NewService(repo Repository, discounts Discounts) *Service {
	return &Service{repo: repo, discounts: discounts}
}

func (s *Service) Create(ctx context.Context, o Order) error {
	if err := o.Validate(); err != nil {
		return err
	}
	o.Total -= o.Total * s.discounts.Percent(o.Code) / 100
	return s.repo.Save(ctx, o)
}

func (s *Service) Get(ctx context.Context, id string) (Order, error) {
	return s.repo.Find(ctx, id)
}
