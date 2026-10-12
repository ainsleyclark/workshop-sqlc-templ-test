package weather

import (
	"context"
	"errors"
	"log/slog"
	"uuid"
)

// Store defines the operations used to persist and query
// weather data.
type Store interface {
	Find(ctx context.Context, id uuid.UUID) (Weather, error)
	Create(ctx context.Context, id uuid.UUID, in CreateParams) (Weather, error)
}

// Service is the entry point for working with weather data.
// Handlers and other domains should use this type instead
// of interacting with the store directly.
type Service struct {
	logger *slog.Logger
	store  Store
}

// NewService returns a Service backed by the given store.
func NewService(
	logger *slog.Logger,
	store Store,
) *Service {
	return &Service{
		logger: logger,
		store:  store,
	}
}

var (
	// ErrNotFound represents an error where no weather has been
	// recorded.
	ErrNotFound = errors.New("weather not found")

	// ErrAlreadyExists represents an error where the weather has
	// already been recorded.
	ErrAlreadyExists = errors.New("weather already exists")

	// ErrInvalid represents an error where a weather record breaks
	// one of the rules in Validate.
	ErrInvalid = errors.New("invalid weather")
)

// Find returns the weather with the given ID, or ErrNotFound.
func (s Service) Find(ctx context.Context, id uuid.UUID) (Weather, error) {
	return s.store.Find(ctx, id)
}

// Create validates the params and assigns the record its identity
// before handing it to the store.
func (s Service) Create(ctx context.Context, params CreateParams) (Weather, error) {
	if err := params.Validate(); err != nil {
		return Weather{}, err
	}

	return s.store.Create(ctx, uuid.New(), params)
}
