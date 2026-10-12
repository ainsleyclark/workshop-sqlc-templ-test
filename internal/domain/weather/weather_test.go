package weather_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
	"uuid"
	"workshop/internal/common/logs"
	"workshop/internal/domain/weather"
	"workshop/internal/domain/weather/stores/weathersqlite"
	"workshop/internal/infra/db/dbtest"
	db "workshop/internal/infra/db/sqlc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errInternal = errors.New("internal error")

func setup(t *testing.T, fail bool) (context.Context, *weather.Service) {
	t.Helper()

	ctx, conn, teardown := dbtest.Setup(t)
	t.Cleanup(teardown)

	q := db.Querier(db.New(conn))
	if fail {
		q = dbtest.FailingQuerier(errInternal)
	}

	return ctx, weather.NewService(
		logs.NewProduction(io.Discard, slog.LevelError),
		weathersqlite.New(q),
	)
}

// newCreateParams returns a fully populated hour's weather.
func newCreateParams() weather.CreateParams {
	return weather.CreateParams{
		ObservedAt:    time.Date(2026, time.September, 22, 9, 0, 0, 0, time.UTC),
		Temperature:   weather.Temperature{Actual: 26.8, Apparent: 29.5},
		Precipitation: weather.Precipitation{Total: 1.2, Rain: 1.2},
		Wind:          weather.Wind{Speed: 24.7, Gusts: 41.3, Direction: 160},
		Condition:     weather.Condition{Code: 61, Description: "Slight rain"},
		CloudCover:    68,
		Humidity:      57,
	}
}

func create(t *testing.T, ctx context.Context, svc *weather.Service, in weather.CreateParams) weather.Weather {
	t.Helper()

	got, err := svc.Create(ctx, in)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil(), got.ID)
	assert.Equal(t, in, got.CreateParams())

	return got
}

func TestService_Find(t *testing.T) {
	t.Run("Found", func(t *testing.T) {
		ctx, svc := setup(t, false)
		created := create(t, ctx, svc, newCreateParams())

		got, err := svc.Find(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created, got)
	})

	t.Run("Not found", func(t *testing.T) {
		ctx, svc := setup(t, false)

		_, err := svc.Find(ctx, uuid.New())
		assert.ErrorIs(t, err, weather.ErrNotFound)
	})

	t.Run("Store error", func(t *testing.T) {
		ctx, svc := setup(t, true)

		_, err := svc.Find(ctx, uuid.New())
		assert.ErrorIs(t, err, errInternal)
	})
}

func TestService_Create(t *testing.T) {
	t.Run("Creates weather", func(t *testing.T) {
		ctx, svc := setup(t, false)
		in := newCreateParams()

		got, err := svc.Create(ctx, in)
		require.NoError(t, err)
		assert.Equal(t, in, got.CreateParams())
		assert.WithinDuration(t, time.Now(), got.CreatedAt, 5*time.Second)
		assert.Equal(t, got.CreatedAt, got.UpdatedAt)
	})

	t.Run("Assigns an identity", func(t *testing.T) {
		ctx, svc := setup(t, false)

		first := create(t, ctx, svc, newCreateParams())
		second := create(t, ctx, svc, newCreateParams())

		assert.NotEqual(t, uuid.Nil(), first.ID)
		assert.NotEqual(t, first.ID, second.ID)
	})

	t.Run("Is retrievable once created", func(t *testing.T) {
		ctx, svc := setup(t, false)
		created := create(t, ctx, svc, newCreateParams())

		got, err := svc.Find(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created, got)
	})

	t.Run("Invalid params are not persisted", func(t *testing.T) {
		ctx, svc := setup(t, true)
		in := newCreateParams()
		in.Humidity = 101

		// The store would fail any call, so ErrInvalid proves it was never reached.
		_, err := svc.Create(ctx, in)
		assert.ErrorIs(t, err, weather.ErrInvalid)
	})

	t.Run("Store error", func(t *testing.T) {
		ctx, svc := setup(t, true)

		_, err := svc.Create(ctx, newCreateParams())
		assert.ErrorIs(t, err, errInternal)
	})
}
